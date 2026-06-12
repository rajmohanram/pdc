package forge

import (
	"context"
	"strings"
	"testing"
)

func TestServiceTree(t *testing.T) {
	nodes, err := ServiceTreeFromSource(context.Background(),
		Options{ProtoPaths: []string{"testdata"}}, TreeOptions{})
	if err != nil {
		t.Fatalf("ServiceTreeFromSource: %v", err)
	}
	// services sorted: "Bare" (no package) before "demo.Greeter"
	if len(nodes) != 2 {
		t.Fatalf("services=%d, want 2", len(nodes))
	}
	if nodes[0].Label != "Bare" {
		t.Errorf("root[0]=%q, want Bare", nodes[0].Label)
	}
	if nodes[1].Label != "demo.Greeter" {
		t.Errorf("root[1]=%q, want demo.Greeter", nodes[1].Label)
	}
	// demo.Greeter has Hello + Health; each method has request + response children
	greeter := nodes[1]
	if len(greeter.Children) != 2 {
		t.Fatalf("Greeter methods=%d, want 2", len(greeter.Children))
	}
	hello := greeter.Children[0]
	if hello.Label != "Hello" {
		t.Errorf("method=%q, want Hello", hello.Label)
	}
	if len(hello.Children) != 2 || hello.Children[0].Label != "request: demo.Req" || hello.Children[1].Label != "response: demo.Resp" {
		t.Errorf("Hello children=%v", hello.Children)
	}
}

func TestServiceTree_Fields(t *testing.T) {
	nodes, err := ServiceTreeFromSource(context.Background(),
		Options{ProtoPaths: []string{"testdata"}}, TreeOptions{Fields: true, Depth: 1})
	if err != nil {
		t.Fatalf("ServiceTreeFromSource: %v", err)
	}
	// demo.Req has "id: string" and "nested: demo.Nested"; at depth 1 the message
	// field is a leaf (not expanded).
	req := nodes[1].Children[0].Children[0] // demo.Greeter -> Hello -> request
	if len(req.Children) != 2 || req.Children[0].Label != "id: string" || req.Children[1].Label != "nested: demo.Nested" {
		t.Fatalf("request fields=%v", req.Children)
	}
	if len(req.Children[1].Children) != 0 {
		t.Errorf("depth 1 should not expand nested, got %v", req.Children[1].Children)
	}
}

func TestServiceTree_Depth(t *testing.T) {
	nodes, err := ServiceTreeFromSource(context.Background(),
		Options{ProtoPaths: []string{"testdata"}}, TreeOptions{Fields: true, Depth: 2})
	if err != nil {
		t.Fatalf("ServiceTreeFromSource: %v", err)
	}
	req := nodes[1].Children[0].Children[0] // demo.Greeter -> Hello -> request
	nested := req.Children[1]               // nested: demo.Nested
	if len(nested.Children) != 1 || nested.Children[0].Label != "note: string" {
		t.Errorf("depth 2 nested fields=%v, want [note: string]", nested.Children)
	}
}

func TestServiceTree_MethodsOnly(t *testing.T) {
	nodes, err := ServiceTreeFromSource(context.Background(),
		Options{ProtoPaths: []string{"testdata"}}, TreeOptions{MethodsOnly: true})
	if err != nil {
		t.Fatalf("ServiceTreeFromSource: %v", err)
	}
	hello := nodes[1].Children[0] // demo.Greeter -> Hello
	if hello.Label != "Hello" || len(hello.Children) != 0 {
		t.Errorf("methods-only Hello should have no children, got %v", hello.Children)
	}
}

func TestServiceTree_ByPackage(t *testing.T) {
	nodes, err := ServiceTreeFromSource(context.Background(),
		Options{ProtoPaths: []string{"testdata"}}, TreeOptions{ByPackage: true})
	if err != nil {
		t.Fatalf("ServiceTreeFromSource: %v", err)
	}
	// roots are packages: "(no package)" and "demo"
	var labels []string
	for _, n := range nodes {
		labels = append(labels, n.Label)
	}
	if got := strings.Join(labels, ","); got != "(no package),demo" {
		t.Fatalf("package roots=%q, want '(no package),demo'", got)
	}
	// under "demo" the service is the short name "Greeter"
	if nodes[1].Children[0].Label != "Greeter" {
		t.Errorf("demo child=%q, want Greeter", nodes[1].Children[0].Label)
	}
}
