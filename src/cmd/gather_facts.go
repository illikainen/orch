package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/illikainen/orch/src/fact"

	"github.com/illikainen/go-utils/src/stringx"
	"github.com/spf13/cobra"
)

var gatherFactsCmd = &cobra.Command{
	Use:    "_gather-facts",
	Short:  "Internal command to gather facts from a remote",
	RunE:   gatherFactsRun,
	Hidden: true,
}

func init() {
	rootCmd.AddCommand(gatherFactsCmd)
}

func gatherFactsRun(cmd *cobra.Command, _ []string) error {
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
