// Package forge compiles .proto sources into a validated, annotated
// FileDescriptorSet for the wasm/authz setup.
//
// Engine: github.com/bufbuild/protocompile (pure Go — no protoc dependency).
// google/api protos are embedded (../../protos); google/protobuf well-known
// types come from protocompile's standard imports.
package forge

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/rajmohanram/pdc/protos"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const annotationsProto = "google/api/annotations.proto"

// Override is a per-method annotation override from the mapping file.
type Override struct {
	Method string `yaml:"method" json:"method"`
	Path   string `yaml:"path" json:"path"`
	Body   string `yaml:"body" json:"body"`
}

// Options configures a generate/inspect run.
type Options struct {
	ProtoPaths     []string
	ProtoFiles     []string // entry files; empty => auto-discover all *.proto under ProtoPaths
	HTTPMethod     string
	Mapping        map[string]Override
	Exclude        []string
	FailOnMissing  bool
	PreferLocalWKT bool

	annotate bool // internal: generate=true, inspect=false
}

// Report summarizes a run.
type Report struct {
	Files       int
	Services    int
	Methods     int
	Annotated   int // methods set to the standard annotation (non-excluded)
	Added       int // of Annotated: previously had no annotation
	Overwritten int // of Annotated: previously had one, now replaced
	Excluded    int
	Missing     []string // methods without an annotation (read/inspect paths)
}

// Generate compiles, sets the standard http annotation on every method
// (overwriting any existing one), and returns the marshaled FileDescriptorSet
// plus a report.
func Generate(ctx context.Context, opts Options) ([]byte, *Report, error) {
	opts.annotate = true
	fdps, rep, err := compileAndAnnotate(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	set := &descriptorpb.FileDescriptorSet{File: fdps}
	out, err := proto.MarshalOptions{Deterministic: true}.Marshal(set)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal descriptor set: %w", err)
	}
	if err := selfCheck(out, opts); err != nil {
		return nil, nil, err
	}
	return out, rep, nil
}

// Inspect compiles from source (without annotating) and reports method status.
func Inspect(ctx context.Context, opts Options) (*Report, error) {
	opts.annotate = false
	_, rep, err := compileAndAnnotate(ctx, opts)
	return rep, err
}

// InspectFile reports method status from a prebuilt descriptor set.
func InspectFile(pb []byte, exclude []string) (*Report, error) {
	set, err := unmarshalSet(pb)
	if err != nil {
		return nil, err
	}
	return reportSet(set, exclude), nil
}

// Validate links a prebuilt descriptor set and reports completeness.
func Validate(pb []byte, exclude []string, failOnMissing bool) (*Report, error) {
	set, err := unmarshalSet(pb)
	if err != nil {
		return nil, err
	}
	if _, err := protodesc.NewFiles(set); err != nil {
		return nil, fmt.Errorf("descriptor set does not link: %w", err)
	}
	rep := reportSet(set, exclude)
	if failOnMissing && len(rep.Missing) > 0 {
		return rep, fmt.Errorf("%d method(s) without an http annotation", len(rep.Missing))
	}
	return rep, nil
}

func compileAndAnnotate(ctx context.Context, opts Options) ([]*descriptorpb.FileDescriptorProto, *Report, error) {
	inputs, err := resolveInputs(opts)
	if err != nil {
		return nil, nil, err
	}
	compiler := protocompile.Compiler{
		Resolver:       buildResolver(opts),
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	linked, err := compiler.Compile(ctx, inputs...)
	if err != nil {
		return nil, nil, fmt.Errorf("compile: %w", err)
	}
	fdps := assemble(linked)
	rep := annotate(fdps, opts)
	return fdps, rep, nil
}

// resolveInputs picks the entry files: explicit --proto-files, or every *.proto
// discovered under --proto-path (so non-imported roots like telemetry.proto are
// never missed). The annotations extension defs are always compiled too.
func resolveInputs(opts Options) ([]string, error) {
	var inputs []string
	if len(opts.ProtoFiles) > 0 {
		inputs = append(inputs, opts.ProtoFiles...)
	} else {
		discovered, err := discover(opts.ProtoPaths)
		if err != nil {
			return nil, err
		}
		inputs = discovered
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no .proto files found under the given --proto-path(s)")
	}
	return appendUnique(inputs, annotationsProto), nil
}

func discover(paths []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		err := filepath.WalkDir(p, func(fp string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(fp, ".proto") {
				return nil
			}
			rel, err := filepath.Rel(p, fp)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if strings.HasPrefix(rel, "google/") { // bundled — resolved, not compiled from user copy
				return nil
			}
			if !seen[rel] {
				seen[rel] = true
				out = append(out, rel)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("discover %q: %w", p, err)
		}
	}
	sort.Strings(out)
	return out, nil
}

func buildResolver(opts Options) protocompile.Resolver {
	embedded := protocompile.ResolverFunc(func(p string) (protocompile.SearchResult, error) {
		f, err := protos.FS.Open(p)
		if err != nil {
			return protocompile.SearchResult{}, err
		}
		return protocompile.SearchResult{Source: f}, nil
	})
	src := &protocompile.SourceResolver{ImportPaths: opts.ProtoPaths}
	var comp protocompile.CompositeResolver
	if opts.PreferLocalWKT {
		comp = protocompile.CompositeResolver{src, embedded}
	} else {
		comp = protocompile.CompositeResolver{embedded, src}
	}
	return protocompile.WithStandardImports(comp)
}

// assemble returns every transitive file as a FileDescriptorProto in dependency
// order (deps before dependents), deduped, deterministic.
func assemble(linked linker.Files) []*descriptorpb.FileDescriptorProto {
	seen := map[string]bool{}
	var out []*descriptorpb.FileDescriptorProto
	var visit func(fd protoreflect.FileDescriptor)
	visit = func(fd protoreflect.FileDescriptor) {
		if seen[fd.Path()] {
			return
		}
		seen[fd.Path()] = true
		imps := fd.Imports()
		names := make([]string, 0, imps.Len())
		byName := map[string]protoreflect.FileDescriptor{}
		for i := 0; i < imps.Len(); i++ {
			imp := imps.Get(i)
			names = append(names, imp.Path())
			byName[imp.Path()] = imp.FileDescriptor
		}
		sort.Strings(names)
		for _, n := range names {
			visit(byName[n])
		}
		out = append(out, proto.Clone(protoutil.ProtoFromFileDescriptor(fd)).(*descriptorpb.FileDescriptorProto))
	}
	tops := make([]protoreflect.FileDescriptor, 0, len(linked))
	for _, f := range linked {
		tops = append(tops, f)
	}
	sort.Slice(tops, func(i, j int) bool { return tops[i].Path() < tops[j].Path() })
	for _, f := range tops {
		visit(f)
	}
	return out
}

func annotate(fdps []*descriptorpb.FileDescriptorProto, opts Options) *Report {
	rep := &Report{Files: len(fdps)}
	for _, fdp := range fdps {
		rep.Services += len(fdp.Service)
		pkg := fdp.GetPackage()
		gained := false
		for _, svc := range fdp.Service {
			svcFull := svc.GetName()
			if pkg != "" {
				svcFull = pkg + "." + svc.GetName()
			}
			for _, m := range svc.Method {
				rep.Methods++
				key := svcFull + "/" + m.GetName()
				had := hasHTTP(m)

				// inspect (read-only): report current state, change nothing.
				if !opts.annotate {
					switch {
					case had:
						rep.Annotated++
					case matchExclude(opts.Exclude, key):
						rep.Excluded++
					default:
						rep.Missing = append(rep.Missing, key)
					}
					continue
				}

				// generate: standardize every non-excluded method, overwriting
				// any pre-existing annotation.
				if matchExclude(opts.Exclude, key) {
					rep.Excluded++
					continue
				}
				method, p, body := resolveAnno(opts, pkg, svc.GetName(), m.GetName(), key)
				setHTTP(m, method, p, body)
				gained = true
				rep.Annotated++
				if had {
					rep.Overwritten++
				} else {
					rep.Added++
				}
			}
		}
		if gained {
			ensureDep(fdp, annotationsProto)
		}
	}
	sort.Strings(rep.Missing)
	return rep
}

// resolveAnno returns the (httpMethod, path, body) for a method: a --mapping
// override if present, else the synthetic standard path.
func resolveAnno(opts Options, pkg, svc, method, key string) (string, string, string) {
	if ov, ok := opts.Mapping[key]; ok {
		hm := ov.Method
		if hm == "" {
			hm = opts.HTTPMethod
		}
		body := ov.Body
		if body == "" && hasBody(hm) {
			body = "*"
		}
		return hm, ov.Path, body
	}
	hm := opts.HTTPMethod
	if hm == "" {
		hm = "post"
	}
	body := ""
	if hasBody(hm) {
		body = "*"
	}
	return hm, syntheticPath(pkg, svc, method), body
}

// syntheticPath builds /<pkg>/<Service>/<Method> with '.' replaced by '/'. A
// missing package becomes the literal segment "pkg" (so the path is always
// /pkg/<Service>/<Method>).
func syntheticPath(pkg, svc, method string) string {
	if pkg == "" {
		pkg = "pkg"
	}
	return "/" + strings.ReplaceAll(pkg+"."+svc+"."+method, ".", "/")
}

func setHTTP(m *descriptorpb.MethodDescriptorProto, httpMethod, p, body string) {
	rule := &annotations.HttpRule{Body: body}
	switch strings.ToLower(httpMethod) {
	case "get":
		rule.Pattern = &annotations.HttpRule_Get{Get: p}
		rule.Body = ""
	case "put":
		rule.Pattern = &annotations.HttpRule_Put{Put: p}
	case "delete":
		rule.Pattern = &annotations.HttpRule_Delete{Delete: p}
		rule.Body = ""
	case "patch":
		rule.Pattern = &annotations.HttpRule_Patch{Patch: p}
	default:
		rule.Pattern = &annotations.HttpRule_Post{Post: p}
	}
	if m.Options == nil {
		m.Options = &descriptorpb.MethodOptions{}
	}
	proto.SetExtension(m.Options, annotations.E_Http, rule)
}

func selfCheck(out []byte, opts Options) error {
	set, err := unmarshalSet(out)
	if err != nil {
		return fmt.Errorf("self-check: %w", err)
	}
	if _, err := protodesc.NewFiles(set); err != nil {
		return fmt.Errorf("self-check: descriptor set does not link: %w", err)
	}
	rep := reportSet(set, opts.Exclude)
	if len(rep.Missing) > 0 && opts.FailOnMissing {
		return fmt.Errorf("self-check: %d method(s) still unannotated: %s",
			len(rep.Missing), strings.Join(rep.Missing, ", "))
	}
	return nil
}

// reportSet builds a Report by reading a raw FileDescriptorSet.
func reportSet(set *descriptorpb.FileDescriptorSet, exclude []string) *Report {
	rep := &Report{Files: len(set.File)}
	for _, fdp := range set.File {
		rep.Services += len(fdp.Service)
		pkg := fdp.GetPackage()
		for _, svc := range fdp.Service {
			svcFull := svc.GetName()
			if pkg != "" {
				svcFull = pkg + "." + svc.GetName()
			}
			for _, m := range svc.Method {
				rep.Methods++
				key := svcFull + "/" + m.GetName()
				switch {
				case hasHTTP(m):
					rep.Annotated++
				case matchExclude(exclude, key):
					rep.Excluded++
				default:
					rep.Missing = append(rep.Missing, key)
				}
			}
		}
	}
	sort.Strings(rep.Missing)
	return rep
}

func hasHTTP(m *descriptorpb.MethodDescriptorProto) bool {
	return m.Options != nil && proto.HasExtension(m.Options, annotations.E_Http)
}

func ensureDep(fdp *descriptorpb.FileDescriptorProto, dep string) {
	for _, d := range fdp.Dependency {
		if d == dep {
			return
		}
	}
	fdp.Dependency = append(fdp.Dependency, dep)
}

func matchExclude(globs []string, key string) bool {
	for _, g := range globs {
		if ok, _ := path.Match(g, key); ok {
			return true
		}
	}
	return false
}

func hasBody(httpMethod string) bool {
	switch strings.ToLower(httpMethod) {
	case "get", "delete":
		return false
	}
	return true
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

func unmarshalSet(pb []byte) (*descriptorpb.FileDescriptorSet, error) {
	var set descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(pb, &set); err != nil {
		return nil, fmt.Errorf("parse FileDescriptorSet: %w", err)
	}
	return &set, nil
}
