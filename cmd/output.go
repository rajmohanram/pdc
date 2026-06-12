package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// outln / outf write to the command's stdout, deliberately ignoring the write
// error (stdout write failures are not actionable here).
func outln(cmd *cobra.Command, a ...any) {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), a...)
}

func outf(cmd *cobra.Command, format string, a ...any) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format, a...)
}
