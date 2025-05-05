package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/orch/src/blueprint"
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/metadata"

	"github.com/illikainen/go-utils/src/flag"
	"github.com/illikainen/go-utils/src/logging"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootOpts struct {
	config    *configs.Config
	configp   flag.Path
	verbosity logging.LogLevel
}

var rootCmd = &cobra.Command{
	Use:               metadata.Name(),
	Version:           fmt.Sprintf("%s (%s@%s)", metadata.Version(), metadata.Branch(), metadata.Commit()),
	PersistentPreRunE: rootPreRun,
}

func Command() *cobra.Command {
	return rootCmd
}

func init() {
	flags := rootCmd.PersistentFlags()
	flags.SortFlags = false

	levels := []string{}
	for _, level := range log.AllLevels {
		levels = append(levels, level.String())
	}

	flags.Var(&rootOpts.configp, "config", "Configuration file")

	lo.Must0(rootOpts.verbosity.Set("info"))
	flags.VarP(&rootOpts.verbosity, "verbosity", "v",
		fmt.Sprintf("Verbosity (%s)", strings.Join(levels, ", ")))

	flags.Bool("help", false, "Help for this command")
}

func rootPreRun(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	config := rootOpts.configp.Value
	if config == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return err
		}

		config = filepath.Join(dir, metadata.Name(), "config.hcl")
	}

	bp := blueprint.NewBlueprint(&blueprint.Options{
		Path: config,
	})

	err := bp.PartialDecode()
	if err != nil {
		return err
	}

	err = bp.Config.Decode(nil)
	if err != nil {
		return err
	}

	rootOpts.config = bp.Config
	return nil
}
