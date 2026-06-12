package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// generateOpts holds the flags for `pdc generate`.
type generateOpts struct {
	protoPaths     []string
	protoFiles     []string
	output         string
	annotate       bool
	httpMethod     string
	pathTemplate   string
	mapping        string
	exclude        []string
	failOnMissing  bool
	dryRun         bool
	preferLocalWKT bool
}

func newGenerateCmd() *cobra.Command {
	o := &generateOpts{}
	c := &cobra.Command{
		Use:   "generate",
		Short: "Compile .proto files (and inject http annotations) into a descriptor set",
		Long: `Compile the given .proto files into a FileDescriptorSet, bundling the
well-known and google.api protos, always including imports and source info.

When --annotate is set (default), methods missing a google.api.http option get
one injected: a per-method override from --mapping, else a synthetic path from
--path-template, unless the method matches an --exclude glob.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.output == "" {
				return errors.New("--output/-o is required")
			}
			if len(o.protoPaths) == 0 {
				return errors.New("at least one --proto-path/-p is required")
			}
			// TODO(pdc): implement the generate pipeline (see DESIGN.md):
			// resolve -> compile (protocompile) -> annotate -> re-validate ->
			// assemble FileDescriptorSet -> deterministic marshal -> self-check.
			return errors.New("generate: not yet implemented — see DESIGN.md")
		},
	}

	f := c.Flags()
	f.StringSliceVarP(&o.protoPaths, "proto-path", "p", nil, "import root(s); repeatable")
	f.StringSliceVarP(&o.protoFiles, "proto-files", "f", nil, "entry .proto file(s); if omitted, all *.proto under --proto-path are discovered")
	f.StringVarP(&o.output, "output", "o", "", "descriptor output file (FileDescriptorSet)")
	f.BoolVar(&o.annotate, "annotate", true, "auto-add google.api.http to methods missing it")
	f.StringVar(&o.httpMethod, "http-method", "post", "HTTP method for synthetic annotations (post|get|put|delete|patch)")
	f.StringVar(&o.pathTemplate, "path-template", "/{pkg}/{service}/{method}", "synthetic path; '.'->'/' applied, {pkg} segment omitted when the service has no package")
	f.StringVar(&o.mapping, "mapping", "", "YAML/JSON file of per-method annotation overrides")
	f.StringSliceVar(&o.exclude, "exclude", nil, "method glob(s) to leave unannotated (e.g. internal-only RPCs)")
	f.BoolVar(&o.failOnMissing, "fail-on-missing", false, "exit non-zero if any non-excluded method ends up unannotated")
	f.BoolVar(&o.dryRun, "dry-run", false, "report what would change without writing the descriptor")
	f.BoolVar(&o.preferLocalWKT, "prefer-local-wkt", false, "use user-supplied google/* protos instead of the bundled ones")
	return c
}
