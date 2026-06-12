package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rajmohanram/pdc/internal/forge"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type generateOpts struct {
	protoPaths     []string
	protoFiles     []string
	output         string
	httpMethod     string
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
		Short: "Compile .proto files into an annotated descriptor set",
		Long: `Compile the given .proto files into a FileDescriptorSet, bundling the
well-known and google.api protos, always including imports and source info.

Every method missing a google.api.http option gets one injected: a per-method
override from --mapping, else a synthetic "/<pkg>/<Service>/<Method>" path,
unless the method matches an --exclude glob.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.output == "" {
				return fmt.Errorf("--output/-o is required")
			}
			if len(o.protoPaths) == 0 {
				return fmt.Errorf("at least one --proto-path/-p is required")
			}
			opts := forge.Options{
				ProtoPaths:     o.protoPaths,
				ProtoFiles:     o.protoFiles,
				HTTPMethod:     o.httpMethod,
				Exclude:        o.exclude,
				FailOnMissing:  o.failOnMissing,
				PreferLocalWKT: o.preferLocalWKT,
			}
			if o.mapping != "" {
				m, err := loadMapping(o.mapping)
				if err != nil {
					return err
				}
				opts.Mapping = m
			}
			out, rep, err := forge.Generate(cmd.Context(), opts)
			if err != nil {
				return err
			}
			outln(cmd, renderReport(rep))
			if o.dryRun {
				outf(cmd, "dry-run: %d bytes not written\n", len(out))
				return nil
			}
			if dir := filepath.Dir(o.output); dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return err
				}
			}
			if err := os.WriteFile(o.output, out, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", o.output, err)
			}
			outf(cmd, "wrote %s (%d bytes)\n", o.output, len(out))
			return nil
		},
	}

	f := c.Flags()
	f.StringSliceVarP(&o.protoPaths, "proto-path", "p", nil, "import root(s); repeatable")
	f.StringSliceVarP(&o.protoFiles, "proto-files", "f", nil, "entry .proto file(s); if omitted, all *.proto under --proto-path are discovered")
	f.StringVarP(&o.output, "output", "o", "", "descriptor output file (FileDescriptorSet)")
	f.StringVar(&o.httpMethod, "http-method", "post", "HTTP method for injected annotations (post|get|put|delete|patch)")
	f.StringVar(&o.mapping, "mapping", "", "YAML/JSON file of per-method annotation overrides")
	f.StringSliceVar(&o.exclude, "exclude", nil, "method glob(s) to leave unannotated (e.g. 'bio.BioService/*')")
	f.BoolVar(&o.failOnMissing, "fail-on-missing", false, "exit non-zero if any non-excluded method ends up unannotated")
	f.BoolVar(&o.dryRun, "dry-run", false, "report what would change without writing the descriptor")
	f.BoolVar(&o.preferLocalWKT, "prefer-local-wkt", false, "use user-supplied google/* protos instead of the bundled ones")
	return c
}

func loadMapping(p string) (map[string]forge.Override, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read mapping: %w", err)
	}
	m := map[string]forge.Override{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse mapping %s: %w", p, err)
	}
	return m, nil
}

func renderReport(r *forge.Report) string {
	return fmt.Sprintf("files=%d services=%d methods=%d annotated=%d (auto=%d, existing=%d) excluded=%d missing=%d",
		r.Files, r.Services, r.Methods, r.Annotated, r.AutoAdded, r.PreExisting, r.Excluded, len(r.Missing))
}
