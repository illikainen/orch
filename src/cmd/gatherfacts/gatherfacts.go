package gatherfactscmd

import (
	"encoding/json"
	"fmt"

	rootcmd "github.com/illikainen/orch/src/cmd/root"
	"github.com/illikainen/orch/src/fact"

	"github.com/illikainen/go-utils/src/stringx"
	"github.com/spf13/cobra"
)

var command = &cobra.Command{
	Use:    "_gather-facts",
	Short:  "Internal command to gather facts from a remote",
	RunE:   run,
	Hidden: true,
}

func Command(_ *rootcmd.Options) *cobra.Command {
	return command
}

func run(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	facts, err := fact.GatherFacts()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(facts, "", "  ")
	if err != nil {
		return err
	}

	_, err = fmt.Printf("%s\n", stringx.Sanitize(data))
	return err
}
