package applycmd

import (
	"github.com/illikainen/orch/src/blueprint"
	rootcmd "github.com/illikainen/orch/src/cmd/root"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/spf13/cobra"
)

var command = &cobra.Command{
	Use:   "apply",
	Short: "Apply a blueprint",
	RunE:  run,
}

var options struct {
	*rootcmd.Options
	file   string
	hosts  []string
	tags   []string
	dryRun bool
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()
	flags.SortFlags = false

	flags.StringVarP(&options.file, "file", "f", "", "Blueprint to apply")
	fn.Must(command.MarkFlagRequired("file"))

	flags.StringSliceVarP(&options.hosts, "host", "h", nil,
		"Only apply on these host(s).  May be provided multiple times")

	flags.StringSliceVarP(&options.tags, "tags", "t", nil,
		"Only apply on hosts with any of these tags(s).  May be provided multiple times")

	flags.BoolVarP(&options.dryRun, "dry-run", "d", false, "Show changes without applying them")
}

func run(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	return blueprint.Apply(&blueprint.Options{
		Path:   options.file,
		Config: options.Config,
		Filter: blueprint.Filter{
			Hosts: options.hosts,
			Tags:  options.tags,
		},
		Sandbox: options.Sandbox,
		DryRun:  options.dryRun,
	})
}
