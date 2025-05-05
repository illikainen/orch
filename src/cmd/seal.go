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

var sealOpts struct {
	input  flag.Path
	output flag.Path
}

var sealCmd = &cobra.Command{
	Use:   "seal",
	Short: "Encrypt and sign a blueprint",
	RunE:  sealRun,
}

func init() {
	flags := sealCmd.Flags()

	sealOpts.input.State = flag.MustExist
	flags.VarP(&sealOpts.input, "input", "i", "Input file to seal")
	lo.Must0(sealCmd.MarkFlagRequired("input"))

	sealOpts.output.State = flag.MustNotExist
	sealOpts.output.Mode = flag.ReadWriteMode
	flags.VarP(&sealOpts.output, "output", "o", "Output file for the sealed blob")
	lo.Must0(sealCmd.MarkFlagRequired("output"))

	rootCmd.AddCommand(sealCmd)
}

func sealRun(cmd *cobra.Command, _ []string) (err error) {
	cmd.SilenceUsage = true

	keys, err := blob.ReadKeyring(rootOpts.config.PrivateKey, rootOpts.config.PublicKeys)
	if err != nil {
		return err
	}

	output, err := os.Create(sealOpts.output.String())
	if err != nil {
		return err
	}
	defer errorx.Defer(output.Close, &err)

	encoder := base64.NewEncoder(base64.StdEncoding.Strict(), output, 72)
	defer errorx.Defer(encoder.Close, &err)

	blobber, err := blob.NewWriter(encoder, &blob.Options{
		Type:      metadata.Name(),
		Keyring:   keys,
		Encrypted: true,
	})
	if err != nil {
		return err
	}
	defer errorx.Defer(blobber.Close, &err)

	input, err := os.Open(sealOpts.input.String())
	if err != nil {
		return err
	}
	defer errorx.Defer(input.Close, &err)

	_, err = io.Copy(blobber, input)
	if err != nil {
		return err
	}

	log.Infof("successfully wrote sealed blueprint to %s", sealOpts.output.String())
	return nil
}
