package rpccmd

import (
	"os"

	rootcmd "github.com/illikainen/orch/src/cmd/root"
	"github.com/illikainen/orch/src/rpc"
	"github.com/illikainen/orch/src/rpc/worker"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var command = &cobra.Command{
	Use:    "_rpc",
	Short:  "Internal command to start RPC worker",
	RunE:   run,
	Hidden: true,
}

func Command(_ *rootcmd.Options) *cobra.Command {
	return command
}

func run(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	log.SetFormatter(&rpc.SanitizedJSONFormatter{})
	log.SetLevel(log.TraceLevel)

	w := worker.New(os.Stdin, os.Stderr)
	err := w.Start()
	if err != nil {
		return err
	}

	return w.Wait()
}
