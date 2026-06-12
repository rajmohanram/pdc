package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pdc",
	Short: "Annotate gRPC methods and forge validated protobuf descriptor sets",
	Long: `pdc compiles .proto files into a self-contained, validated
FileDescriptorSet for the Envoy WASM authz filter.

It bundles the well-known and google.api protos, always includes imports and
source info, and can inject google.api.http annotations on methods that lack
them — producing a descriptor that the wasm/authz pipeline can fully inspect.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		newGenerateCmd(),
		newInspectCmd(),
		newValidateCmd(),
		newVersionCmd(),
	)
}
