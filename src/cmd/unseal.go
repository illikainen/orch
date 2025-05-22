package cmd

import (
	"io"
	"os"

	"github.com/illikainen/orch/src/metadata"

	"github.com/illikainen/go-cryptor/src/blob"
	"github.com/illikainen/go-utils/src/base64"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/flag"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var unsealOpts struct {
	input  flag.Path
	output flag.Path
}

var unsealCmd = &cobra.Command{
	Use:   "unseal",
	Short: "Verify and decrypt a blueprint",
	RunE:  unsealRun,
}

func init() {
	flags := unsealCmd.Flags()

	unsealOpts.input.State = flag.MustExist
	flags.VarP(&unsealOpts.input, "input", "i", "File to unseal")
	lo.Must0(unsealCmd.MarkFlagRequired("input"))

	unsealOpts.output.State = flag.MustBeDir
	unsealOpts.output.Mode = flag.ReadWriteMode
	flags.VarP(&unsealOpts.output, "output", "o", "Output file for the unsealed blob")
	lo.Must0(unsealCmd.MarkFlagRequired("output"))

	rootCmd.AddCommand(unsealCmd)
}

func unsealRun(cmd *cobra.Command, _ []string) (err error) {
	cmd.SilenceUsage = true

	keys, err := blob.ReadKeyring(rootOpts.config.PrivateKey, rootOpts.config.PublicKeys)
	if err != nil {
		return err
	}

	input, err := os.Open(unsealOpts.input.String())
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

	output, err := os.Create(unsealOpts.output.String())
	if err != nil {
		return err
	}
	defer errorx.Defer(output.Close, &err)

	_, err = io.Copy(output, blobber)
	if err != nil {
		return err
	}

	log.Infof("successfully wrote unsealed blueprint to %s", unsealOpts.output.String())
	return nil
}
