package cmd

import (
	applycmd "github.com/illikainen/orch/src/cmd/apply"
	applytaskcmd "github.com/illikainen/orch/src/cmd/applytask"
	gatherfactscmd "github.com/illikainen/orch/src/cmd/gatherfacts"
	genkeycmd "github.com/illikainen/orch/src/cmd/genkey"
	rootcmd "github.com/illikainen/orch/src/cmd/root"
	sealcmd "github.com/illikainen/orch/src/cmd/seal"
	unsealcmd "github.com/illikainen/orch/src/cmd/unseal"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	c, opts := rootcmd.Command()
	c.AddCommand(applycmd.Command(opts))
	c.AddCommand(applytaskcmd.Command(opts))
	c.AddCommand(gatherfactscmd.Command(opts))
	c.AddCommand(genkeycmd.Command(opts))
	c.AddCommand(sealcmd.Command(opts))
	c.AddCommand(unsealcmd.Command(opts))
	return c
}
