package forge

import (
	"bytes"
	"context"
	"testing"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestResolveAnno(t *testing.T) {
	cases := []struct {
		name, pkg, svc, method, httpMethod string
		wantMethod, wantPath, wantBody     string
	}{
		{"with-pkg", "demo", "Greeter", "Hello", "post", "post", "/demo/Greeter/Hello", "*"},
		{"no-pkg-uses-literal-pkg", "", "Bare", "Beat", "post", "post", "/pkg/Bare/Beat", "*"},
		{"multi-segment-pkg", "a.b", "S", "M", "post", "post", "/a/b/S/M", "*"},
		{"get-has-no-body", "demo", "Greeter", "Hello", "get", "get", "/demo/Greeter/Hello", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, p, b := resolveAnno(Options{HTTPMethod: c.httpMethod}, c.pkg, c.svc, c.method, c.svc+"/"+c.method)
			if m != c.wantMethod || p != c.wantPath || b != c.wantBody {
				t.Fatalf("got (%q,%q,%q), want (%q,%q,%q)", m, p, b, c.wantMethod, c.wantPath, c.wantBody)
			}
		})
	}
}

func TestResolveAnno_MappingOverride(t *testing.T) {
	opts := Options{HTTPMethod: "post", Mapping: map[string]Override{
		"demo.Greeter/Hello": {Method: "get", Path: "/v1/hello"},
	}}
	m, p, b := resolveAnno(opts, "demo", "Greeter", "Hello", "demo.Greeter/Hello")
	if m != "get" || p != "/v1/hello" || b != "" {
		t.Fatalf("override got (%q,%q,%q)", m, p, b)
	}
}

func generateSet(t *testing.T, opts Options) (*descriptorpb.FileDescriptorSet, *Report, []byte) {
	t.Helper()
	if opts.ProtoPaths == nil {
		opts.ProtoPaths = []string{"testdata"}
	}
	if opts.HTTPMethod == "" {
		opts.HTTPMethod = "post"
	}
	out, rep, err := Generate(context.Background(), opts)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var set descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(out, &set); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &set, rep, out
}

func TestGenerate_AnnotatesAll(t *testing.T) {
	set, rep, _ := generateSet(t, Options{})

	if rep.Methods != 3 {
		t.Errorf("methods=%d, want 3", rep.Methods)
	}
	if rep.Added != 2 {
		t.Errorf("added=%d, want 2 (Hello, Beat)", rep.Added)
	}
	if rep.Overwritten != 1 {
		t.Errorf("overwritten=%d, want 1 (Health)", rep.Overwritten)
	}
	if rep.Annotated != 3 {
		t.Errorf("annotated=%d, want 3", rep.Annotated)
	}
	if len(rep.Missing) != 0 {
		t.Errorf("missing=%v, want none", rep.Missing)
	}
	if _, err := protodesc.NewFiles(set); err != nil {
		t.Fatalf("descriptor set does not link: %v", err)
	}
	// auto-discovery picked up the non-imported root; no package -> literal "pkg"
	if got := httpPath(t, set, "Bare", "Beat"); got != "/pkg/Bare/Beat" {
		t.Errorf("Beat path=%q, want /pkg/Bare/Beat", got)
	}
	// synthetic path for a packaged service
	if got := httpPath(t, set, "demo.Greeter", "Hello"); got != "/demo/Greeter/Hello" {
		t.Errorf("Hello path=%q, want /demo/Greeter/Hello", got)
	}
	// pre-existing annotation overwritten to the standard path
	if got := httpPath(t, set, "demo.Greeter", "Health"); got != "/demo/Greeter/Health" {
		t.Errorf("Health path=%q, want /demo/Greeter/Health (overwritten)", got)
	}
}

func TestGenerate_Deterministic(t *testing.T) {
	_, _, a := generateSet(t, Options{})
	_, _, b := generateSet(t, Options{})
	if !bytes.Equal(a, b) {
		t.Fatal("output not byte-identical across runs")
	}
}

func TestGenerate_Exclude(t *testing.T) {
	_, rep, _ := generateSet(t, Options{Exclude: []string{"demo.Greeter/Hello"}})
	if rep.Excluded != 1 {
		t.Errorf("excluded=%d, want 1", rep.Excluded)
	}
	if rep.Added != 1 {
		t.Errorf("added=%d, want 1 (Beat)", rep.Added)
	}
	if rep.Overwritten != 1 {
		t.Errorf("overwritten=%d, want 1 (Health)", rep.Overwritten)
	}
	if len(rep.Missing) != 0 {
		t.Errorf("missing=%v, want none (excluded is not missing)", rep.Missing)
	}
}

// httpPath returns the http rule path for a method, or "" if unannotated.
func httpPath(t *testing.T, set *descriptorpb.FileDescriptorSet, svcFull, method string) string {
	t.Helper()
	for _, f := range set.File {
		pkg := f.GetPackage()
		for _, s := range f.Service {
			full := s.GetName()
			if pkg != "" {
				full = pkg + "." + s.GetName()
			}
			if full != svcFull {
				continue
			}
			for _, m := range s.Method {
				if m.GetName() != method {
					continue
				}
				if m.Options == nil || !proto.HasExtension(m.Options, annotations.E_Http) {
					return ""
				}
				rule := proto.GetExtension(m.Options, annotations.E_Http).(*annotations.HttpRule)
				switch p := rule.Pattern.(type) {
				case *annotations.HttpRule_Post:
					return p.Post
				case *annotations.HttpRule_Get:
					return p.Get
				case *annotations.HttpRule_Put:
					return p.Put
				case *annotations.HttpRule_Delete:
					return p.Delete
				case *annotations.HttpRule_Patch:
					return p.Patch
				}
			}
		}
	}
	return ""
}
