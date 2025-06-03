package rootcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/orch/src/blueprint"
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/metadata"

	"github.com/illikainen/go-utils/src/process"
	"github.com/illikainen/go-utils/src/sandbox"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type Options struct {
	Config    *configs.Config
	config    string
	Sandbox   sandbox.Sandbox
	sandbox   string
	Verbosity string
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

	flags.StringVarP(&options.config, "config", "",
		filepath.Join(lo.Must1(os.UserConfigDir()), metadata.Name(), "config.hcl"),
		"Configuration file")

	flags.StringVarP(&options.sandbox, "sandbox", "", "", "Sandbox backend")

	levels := []string{}
	for _, level := range log.AllLevels {
		levels = append(levels, level.String())
	}
	flags.StringVarP(&options.Verbosity, "verbosity", "V", "info",
		fmt.Sprintf("Verbosity (%s)", strings.Join(levels, ", ")))

	flags.Bool("help", false, "Help for this command")
}

func preRun(_ *cobra.Command, _ []string) error {
	level, err := log.ParseLevel(options.Verbosity)
	if err != nil {
		return err
	}
	log.SetLevel(level)

	bp := blueprint.NewBlueprint(&blueprint.Options{
		Path:         options.config,
		AllowMissing: true,
	})

	err = bp.PartialDecode()
	if err != nil {
		return err
	}

	err = bp.Config.Decode(nil)
	if err != nil {
		return err
	}

	options.Config = bp.Config

	name := lo.Ternary(options.sandbox != "", options.sandbox, options.Config.Sandbox)
	backend, err := sandbox.Backend(name)
	if err != nil {
		return err
	}

	switch backend {
	case sandbox.BubblewrapSandbox:
		options.Sandbox, err = sandbox.NewBubblewrap(&sandbox.BubblewrapOptions{
			ReadOnlyPaths: append([]string{
				options.config,
				options.Config.PrivateKey,
			}, options.Config.PublicKeys...),
			ReadWritePaths:   []string{},
			Tmpfs:            true,
			Devtmpfs:         true,
			Procfs:           true,
			AllowCommonPaths: true,
			Stdout:           process.LogrusOutput,
			Stderr:           process.LogrusOutput,
		})
		if err != nil {
			return err
		}
	case sandbox.NoSandbox:
		options.Sandbox, err = sandbox.NewNoop()
		if err != nil {
			return err
		}
	}

	return nil
}
