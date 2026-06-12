package forge

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
)

// Node is a generic tree node for rendering.
type Node struct {
	Label    string
	Children []Node
}

// TreeOptions configures the service tree.
type TreeOptions struct {
	Fields      bool // expand each request/response message's fields
	Depth       int  // with Fields: levels of nested message fields (<=0 = unlimited)
	ByPackage   bool // group services under their package as the top root
	MethodsOnly bool // compact: services and methods only, no message nodes
}

// ServiceTreeFromSource compiles the protos and builds the service tree.
func ServiceTreeFromSource(ctx context.Context, opts Options, topt TreeOptions) ([]Node, error) {
	opts.annotate = false
	fdps, _, err := compileAndAnnotate(ctx, opts)
	if err != nil {
		return nil, err
	}
	return serviceTree(&descriptorpb.FileDescriptorSet{File: fdps}, topt), nil
}

// ServiceTreeFromBytes builds the service tree from a prebuilt descriptor set.
func ServiceTreeFromBytes(pb []byte, topt TreeOptions) ([]Node, error) {
	set, err := unmarshalSet(pb)
	if err != nil {
		return nil, err
	}
	return serviceTree(set, topt), nil
}

// serviceTree builds a [package ->] service -> method -> request/response[-> fields] forest.
func serviceTree(set *descriptorpb.FileDescriptorSet, topt TreeOptions) []Node {
	idx := messageIndex(set)

	type svc struct {
		pkg   string
		short string
		fd    *descriptorpb.ServiceDescriptorProto
	}
	var svcs []svc
	for _, f := range set.File {
		for _, s := range f.Service {
			svcs = append(svcs, svc{f.GetPackage(), s.GetName(), s})
		}
	}
	sort.Slice(svcs, func(i, j int) bool {
		if svcs[i].pkg != svcs[j].pkg {
			return svcs[i].pkg < svcs[j].pkg
		}
		return svcs[i].short < svcs[j].short
	})

	methods := func(fd *descriptorpb.ServiceDescriptorProto) []Node {
		var ms []Node
		for _, m := range fd.Method {
			mn := Node{Label: methodLabel(m)}
			if !topt.MethodsOnly {
				mn.Children = append(mn.Children,
					messageNode("request", strings.TrimPrefix(m.GetInputType(), "."), m.GetClientStreaming(), idx, topt),
					messageNode("response", strings.TrimPrefix(m.GetOutputType(), "."), m.GetServerStreaming(), idx, topt),
				)
			}
			ms = append(ms, mn)
		}
		return ms
	}

	if topt.ByPackage {
		var order []string
		groups := map[string][]svc{}
		for _, s := range svcs {
			key := s.pkg
			if key == "" {
				key = "(no package)"
			}
			if _, ok := groups[key]; !ok {
				order = append(order, key)
			}
			groups[key] = append(groups[key], s)
		}
		sort.Strings(order)
		roots := make([]Node, 0, len(order))
		for _, pkg := range order {
			pn := Node{Label: pkg}
			for _, s := range groups[pkg] {
				pn.Children = append(pn.Children, Node{Label: s.short, Children: methods(s.fd)})
			}
			roots = append(roots, pn)
		}
		return roots
	}

	roots := make([]Node, 0, len(svcs))
	for _, s := range svcs {
		full := s.short
		if s.pkg != "" {
			full = s.pkg + "." + s.short
		}
		roots = append(roots, Node{Label: full, Children: methods(s.fd)})
	}
	return roots
}

func methodLabel(m *descriptorpb.MethodDescriptorProto) string {
	switch {
	case m.GetClientStreaming() && m.GetServerStreaming():
		return m.GetName() + "  (bidi stream)"
	case m.GetClientStreaming():
		return m.GetName() + "  (client stream)"
	case m.GetServerStreaming():
		return m.GetName() + "  (server stream)"
	default:
		return m.GetName()
	}
}

func messageNode(role, msgFull string, stream bool, idx map[string]*descriptorpb.DescriptorProto, topt TreeOptions) Node {
	display := msgFull
	if stream {
		display = "stream " + msgFull
	}
	n := Node{Label: role + ": " + display}
	if !topt.Fields {
		return n
	}
	levels := topt.Depth
	if levels == 0 {
		levels = -1 // 0 => unlimited
	}
	n.Children = fieldNodes(msgFull, idx, levels, map[string]bool{msgFull: true})
	return n
}

// fieldNodes lists a message's fields, recursing into message-typed fields up to
// `levels` deep (levels < 0 = unlimited). onPath guards against type cycles.
func fieldNodes(msgFull string, idx map[string]*descriptorpb.DescriptorProto, levels int, onPath map[string]bool) []Node {
	msg, ok := idx[msgFull]
	if !ok {
		return nil
	}
	var out []Node
	for _, f := range msg.Field {
		fn := Node{Label: fieldLabel(f)}
		if f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE && (levels < 0 || levels > 1) {
			child := strings.TrimPrefix(f.GetTypeName(), ".")
			if child != "" && !onPath[child] {
				onPath[child] = true
				fn.Children = fieldNodes(child, idx, nextLevel(levels), onPath)
				delete(onPath, child)
			}
		}
		out = append(out, fn)
	}
	return out
}

func nextLevel(levels int) int {
	if levels < 0 {
		return -1
	}
	return levels - 1
}

func fieldLabel(f *descriptorpb.FieldDescriptorProto) string {
	typ := fieldTypeStr(f)
	if f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		typ = "repeated " + typ
	}
	return fmt.Sprintf("%s: %s", f.GetName(), typ)
}

func fieldTypeStr(f *descriptorpb.FieldDescriptorProto) string {
	if tn := f.GetTypeName(); tn != "" {
		return strings.TrimPrefix(tn, ".")
	}
	return strings.ToLower(strings.TrimPrefix(f.GetType().String(), "TYPE_"))
}

// messageIndex maps fully-qualified message name -> descriptor (incl. nested).
func messageIndex(set *descriptorpb.FileDescriptorSet) map[string]*descriptorpb.DescriptorProto {
	idx := map[string]*descriptorpb.DescriptorProto{}
	var add func(prefix string, msgs []*descriptorpb.DescriptorProto)
	add = func(prefix string, msgs []*descriptorpb.DescriptorProto) {
		for _, m := range msgs {
			full := m.GetName()
			if prefix != "" {
				full = prefix + "." + m.GetName()
			}
			idx[full] = m
			add(full, m.NestedType)
		}
	}
	for _, f := range set.File {
		add(f.GetPackage(), f.MessageType)
	}
	return idx
}
