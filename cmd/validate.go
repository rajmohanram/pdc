package cmd

import (
	"fmt"
	"os"

	"github.com/rajmohanram/pdc/internal/forge"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var (
		input         string
		exclude       []string
		failOnMissing bool
	)
	c := &cobra.Command{
		Use:   "validate",
		Short: "Validate that a descriptor set links and is authz-ready",
		Long: `Load a prebuilt FileDescriptorSet, assert it links (imports fully
resolve, types self-contained), and report annotation coverage. With
--fail-on-missing, exit non-zero if any non-excluded method lacks an annotation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" {
				return fmt.Errorf("--input/-i <descriptor.pb> is required")
			}
			pb, err := os.ReadFile(input)
			if err != nil {
				return err
			}
			rep, err := forge.Validate(pb, exclude, failOnMissing)
			if rep != nil {
				fmt.Fprintln(cmd.OutOrStdout(), renderReport(rep))
			}
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "OK: descriptor links and is complete")
			return nil
		},
	}
	f := c.Flags()
	f.StringVarP(&input, "input", "i", "", "FileDescriptorSet to validate")
	f.StringSliceVar(&exclude, "exclude", nil, "method glob(s) allowed to be unannotated")
	f.BoolVar(&failOnMissing, "fail-on-missing", true, "fail if any non-excluded method lacks an http annotation")
	return c
}
