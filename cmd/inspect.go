package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rajmohanram/pdc/internal/forge"
)

func newInspectCmd() *cobra.Command {
	var (
		input       string
		protoPaths  []string
		protoFiles  []string
		exclude     []string
		missingOnly bool
	)
	c := &cobra.Command{
		Use:   "inspect",
		Short: "List method annotation status (what the filter would deny)",
		Long: `Report every service/method and whether it carries a google.api.http
annotation. Reads a prebuilt descriptor (--input) or compiles from source
(--proto-path/--proto-files). --missing-only lists just the unannotated methods.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				rep *forge.Report
				err error
			)
			if input != "" {
				pb, e := os.ReadFile(input)
				if e != nil {
					return e
				}
				rep, err = forge.InspectFile(pb, exclude)
			} else {
				if len(protoPaths) == 0 {
					return fmt.Errorf("provide --input <descriptor.pb> or --proto-path")
				}
				rep, err = forge.Inspect(cmd.Context(), forge.Options{
					ProtoPaths: protoPaths,
					ProtoFiles: protoFiles,
					Exclude:    exclude,
				})
			}
			if err != nil {
				return err
			}
			outln(cmd, renderReport(rep))
			if missingOnly {
				for _, m := range rep.Missing {
					outln(cmd, "  MISSING", m)
				}
			}
			return nil
		},
	}
	f := c.Flags()
	f.StringVarP(&input, "input", "i", "", "prebuilt FileDescriptorSet to inspect")
	f.StringSliceVarP(&protoPaths, "proto-path", "p", nil, "import root(s) (when compiling from source)")
	f.StringSliceVarP(&protoFiles, "proto-files", "f", nil, "entry .proto file(s)")
	f.StringSliceVar(&exclude, "exclude", nil, "method glob(s) to treat as intentionally unannotated")
	f.BoolVar(&missingOnly, "missing-only", false, "list only methods without an http annotation")
	return c
}
