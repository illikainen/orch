package rootcmd

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

type Options struct {
	Config    *configs.Config
	Configp   flag.Path
	Verbosity logging.LogLevel
}

var options = Options{}

var command = &cobra.Command{
	Use:               metadata.Name(),
	Version:           fmt.Sprintf("%s (%s@%s)", metadata.Version(), metadata.Branch(), metadata.Commit()),
	PersistentPreRunE: preRun,
}

func Command() (*cobra.Command, *Options) {
	return command, &options
}

func init() {
	flags := command.PersistentFlags()
	flags.SortFlags = false

	levels := []string{}
	for _, level := range log.AllLevels {
		levels = append(levels, level.String())
	}

	flags.Var(&options.Configp, "config", "Configuration file")

	lo.Must0(options.Verbosity.Set("info"))
	flags.VarP(&options.Verbosity, "verbosity", "v",
		fmt.Sprintf("Verbosity (%s)", strings.Join(levels, ", ")))

	flags.Bool("help", false, "Help for this command")
}

func preRun(_ *cobra.Command, _ []string) error {
	config := options.Configp.Value
	if config == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return err
		}

		config = filepath.Join(dir, metadata.Name(), "config.hcl")
	}

	bp := blueprint.NewBlueprint(&blueprint.Options{
		Path:         config,
		AllowMissing: true,
	})

	err := bp.PartialDecode()
	if err != nil {
		return err
	}

	err = bp.Config.Decode(nil)
	if err != nil {
		return err
	}

	options.Config = bp.Config
	return nil
}
