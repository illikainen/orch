package cmd

import (
	applycmd "github.com/illikainen/orch/src/cmd/apply"
	genkeycmd "github.com/illikainen/orch/src/cmd/genkey"
	rootcmd "github.com/illikainen/orch/src/cmd/root"
	rpccmd "github.com/illikainen/orch/src/cmd/rpc"
	sealcmd "github.com/illikainen/orch/src/cmd/seal"
	unsealcmd "github.com/illikainen/orch/src/cmd/unseal"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	c, opts := rootcmd.Command()
	c.AddCommand(applycmd.Command(opts))
	c.AddCommand(genkeycmd.Command(opts))
	c.AddCommand(rpccmd.Command(opts))
	c.AddCommand(sealcmd.Command(opts))
	c.AddCommand(unsealcmd.Command(opts))
	return c
}
