package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var (
		input         string
		failOnMissing bool
	)
	c := &cobra.Command{
		Use:   "validate",
		Short: "Validate that a descriptor set is complete and authz-ready",
		Long: `Load a prebuilt FileDescriptorSet and assert it is sound for the
wasm/authz pipeline: imports fully resolved, source info present, types
self-contained, and (optionally) every method annotated.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" {
				return errors.New("--input/-i <descriptor.pb> is required")
			}
			// TODO(descforge): load, re-link, and assert invariants.
			return errors.New("validate: not yet implemented — see DESIGN.md")
		},
	}
	f := c.Flags()
	f.StringVarP(&input, "input", "i", "", "FileDescriptorSet to validate")
	f.BoolVar(&failOnMissing, "fail-on-missing", true, "fail if any method lacks an http annotation")
	return c
}
