package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rajmohanram/pdc/internal/forge"
	"github.com/spf13/cobra"
)

func newTreeCmd() *cobra.Command {
	var (
		input       string
		protoPaths  []string
		protoFiles  []string
		fields      bool
		depth       int
		byPackage   bool
		methodsOnly bool
	)
	c := &cobra.Command{
		Use:   "tree",
		Short: "Print a tree of services, methods, and messages",
		Long: `Render services as a tree of their methods and request/response message
types. Reads a prebuilt descriptor (--input) or compiles from source
(--proto-path/--proto-files).

  --fields        also list each message's fields
  --depth N       expand N levels of nested message fields (0 = unlimited; implies --fields)
  --by-package    group services under their package as the top root
  --methods-only  compact: services and methods only, no message nodes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --depth implies --fields.
			if cmd.Flags().Changed("depth") {
				fields = true
			}
			topt := forge.TreeOptions{
				Fields:      fields,
				Depth:       depth,
				ByPackage:   byPackage,
				MethodsOnly: methodsOnly,
			}
			var (
				nodes []forge.Node
				err   error
			)
			if input != "" {
				pb, e := os.ReadFile(input)
				if e != nil {
					return e
				}
				nodes, err = forge.ServiceTreeFromBytes(pb, topt)
			} else {
				if len(protoPaths) == 0 {
					return fmt.Errorf("provide --input <descriptor.pb> or --proto-path")
				}
				nodes, err = forge.ServiceTreeFromSource(cmd.Context(), forge.Options{
					ProtoPaths: protoPaths,
					ProtoFiles: protoFiles,
				}, topt)
			}
			if err != nil {
				return err
			}
			if len(nodes) == 0 {
				outln(cmd, "(no services)")
				return nil
			}
			outln(cmd, renderForest(nodes))
			return nil
		},
	}
	f := c.Flags()
	f.StringVarP(&input, "input", "i", "", "prebuilt FileDescriptorSet to read")
	f.StringSliceVarP(&protoPaths, "proto-path", "p", nil, "import root(s) (when compiling from source)")
	f.StringSliceVarP(&protoFiles, "proto-files", "f", nil, "entry .proto file(s)")
	f.BoolVar(&fields, "fields", false, "expand each message's fields")
	f.IntVar(&depth, "depth", 1, "with --fields: levels of nested message fields to expand (0 = unlimited)")
	f.BoolVar(&byPackage, "by-package", false, "group services under their package as the top root")
	f.BoolVar(&methodsOnly, "methods-only", false, "compact: services and methods only, no message nodes")
	return c
}

// renderForest renders top-level nodes flush-left and descendants with tree
// connectors.
func renderForest(roots []forge.Node) string {
	var b strings.Builder
	for _, r := range roots {
		b.WriteString(r.Label + "\n")
		renderChildren(&b, r.Children, "")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderChildren(b *strings.Builder, nodes []forge.Node, prefix string) {
	for i, n := range nodes {
		last := i == len(nodes)-1
		conn, childPrefix := "├─ ", prefix+"│  "
		if last {
			conn, childPrefix = "└─ ", prefix+"   "
		}
		b.WriteString(prefix + conn + n.Label + "\n")
		renderChildren(b, n.Children, childPrefix)
	}
}
