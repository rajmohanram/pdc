package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	var (
		input        string
		protoPaths   []string
		protoFiles   []string
		missingOnly  bool
	)
	c := &cobra.Command{
		Use:   "inspect",
		Short: "List services/methods and their google.api.http annotation status",
		Long: `Report every service and method and whether it carries a google.api.http
annotation. Reads a prebuilt descriptor (--input) or compiles from source
(--proto-path/--proto-files). Use --missing-only to list just the methods the
wasm filter would deny in fail-closed mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" && len(protoPaths) == 0 {
				return errors.New("provide --input <descriptor.pb> or --proto-path")
			}
			// TODO(pdc): load descriptor and print the annotation report.
			return errors.New("inspect: not yet implemented — see DESIGN.md")
		},
	}
	f := c.Flags()
	f.StringVarP(&input, "input", "i", "", "prebuilt FileDescriptorSet to inspect")
	f.StringSliceVarP(&protoPaths, "proto-path", "p", nil, "import root(s) (when compiling from source)")
	f.StringSliceVarP(&protoFiles, "proto-files", "f", nil, "entry .proto file(s)")
	f.BoolVar(&missingOnly, "missing-only", false, "list only methods without an http annotation")
	return c
}
