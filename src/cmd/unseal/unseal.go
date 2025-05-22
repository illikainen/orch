package unsealcmd

import (
	"io"
	"os"

	rootcmd "github.com/illikainen/orch/src/cmd/root"
	"github.com/illikainen/orch/src/metadata"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/base64"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/flag"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var options struct {
	*rootcmd.Options
	input  flag.Path
	output flag.Path
}

var command = &cobra.Command{
	Use:   "unseal",
	Short: "Verify and decrypt a blueprint",
	RunE:  run,
}

func Command(opts *rootcmd.Options) *cobra.Command {
	options.Options = opts
	return command
}

func init() {
	flags := command.Flags()

	options.input.State = flag.MustExist
	flags.VarP(&options.input, "input", "i", "File to unseal")
	lo.Must0(command.MarkFlagRequired("input"))

	options.output.State = flag.MustBeDir
	options.output.Mode = flag.ReadWriteMode
	flags.VarP(&options.output, "output", "o", "Output file for the unsealed blob")
	lo.Must0(command.MarkFlagRequired("output"))
}

func run(cmd *cobra.Command, _ []string) (err error) {
	cmd.SilenceUsage = true

	keys, err := blob.ReadKeyring(options.Config.PrivateKey, options.Config.PublicKeys)
	if err != nil {
		return err
	}

	input, err := os.Open(options.input.String())
	if err != nil {
		return err
	}
	defer errorx.Defer(input.Close, &err)

	decoder, err := base64.NewDecoder(base64.StdEncoding.Strict(), input)
	if err != nil {
		return err
	}

	blobber, err := blob.NewReader(decoder, &blob.Options{
		Type:      metadata.Name(),
		Keyring:   keys,
		Encrypted: true,
	})
	if err != nil {
		return err
	}

	output, err := os.Create(options.output.String())
	if err != nil {
		return err
	}
	defer errorx.Defer(output.Close, &err)

	_, err = io.Copy(output, blobber)
	if err != nil {
		return err
	}

	log.Infof("successfully wrote unsealed blueprint to %s", options.output.String())
	return nil
}
