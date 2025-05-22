package cmd

import (
	"github.com/illikainen/orch/src/blueprint"

	"github.com/illikainen/go-utils/src/flag"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var applyOpts struct {
	file   flag.Path
	hosts  []string
	tags   []string
	dryRun bool
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a blueprint",
	RunE:  applyRun,
}

func init() {
	flags := applyCmd.Flags()
	flags.SortFlags = false

	applyOpts.file.State = flag.MustExist | flag.MustBeFile
	flags.VarP(&applyOpts.file, "file", "f", "Blueprint to apply")
	lo.Must0(applyCmd.MarkFlagRequired("file"))

	flags.StringSliceVarP(&applyOpts.hosts, "host", "h", nil,
		"Only apply on these host(s).  May be provided multiple times")

	flags.StringSliceVarP(&applyOpts.tags, "tags", "t", nil,
		"Only apply on hosts with any of these tags(s).  May be provided multiple times")

	flags.BoolVarP(&applyOpts.dryRun, "dry-run", "d", false, "Show changes without applying them")

	rootCmd.AddCommand(applyCmd)
}

func applyRun(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	return blueprint.Apply(&blueprint.Options{
		Path:   applyOpts.file.Value,
		Config: rootOpts.config,
		Filter: blueprint.Filter{
			Hosts: applyOpts.hosts,
			Tags:  applyOpts.tags,
		},
		DryRun: applyOpts.dryRun,
	})
}
