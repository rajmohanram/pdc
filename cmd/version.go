package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Set via -ldflags at build time (see Makefile).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("descforge %s (commit %s, built %s)\n", version, commit, date)
		},
	}
}
