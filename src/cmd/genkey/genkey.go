package genkeycmd

import (
	"fmt"
	"time"

	rootcmd "github.com/illikainen/orch/src/cmd/root"

	"github.com/illikainen/go-cryptor/src/asymmetric"
	"github.com/illikainen/go-utils/src/fn"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var options struct {
	output string
	delay  time.Duration
}

var command = &cobra.Command{
	Use:   "genkey",
	Short: "Generate a keypair",
	RunE:  run,
}

func Command(_ *rootcmd.Options) *cobra.Command {
	return command
}

func init() {
	flags := command.Flags()

	flags.StringVarP(&options.output, "output", "o", "",
		"Write the generated keypair to <output>.pub and <output>.priv")
	fn.Must(command.MarkFlagRequired("output"))

	flags.DurationVarP(&options.delay, "delay", "d", 60*time.Second,
		"Add a delay between each generated key")
}

func run(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	pubKey, privKey, err := asymmetric.GenerateKey(options.delay)
	if err != nil {
		return err
	}

	pubFile := fmt.Sprintf("%s.pub", options.output)
	err = pubKey.Write(pubFile)
	if err != nil {
		return err
	}

	privFile := fmt.Sprintf("%s.priv", options.output)
	err = privKey.Write(privFile)
	if err != nil {
		return err
	}

	log.Infof("successfully wrote %s to %s and %s", pubKey.Fingerprint(), pubFile, privFile)
	return nil
}
