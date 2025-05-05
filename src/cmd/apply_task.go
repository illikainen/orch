package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/illikainen/go-utils/src/logging"
	log "github.com/sirupsen/logrus"

	"github.com/illikainen/orch/src/tasks"

	"github.com/illikainen/go-utils/src/stringx"
	"github.com/spf13/cobra"
)

var applyTaskCmd = &cobra.Command{
	Use:    "_apply-task",
	Short:  "Internal command to apply a task",
	RunE:   applyTaskRun,
	Hidden: true,
}

func init() {
	rootCmd.AddCommand(applyTaskCmd)
}

func applyTaskRun(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	log.SetFormatter(&logging.SanitizedJSONFormatter{})

	data := bytes.Buffer{}
	_, err := io.Copy(&data, os.Stdin)
	if err != nil {
		return err
	}

	task := &tasks.Task{}
	err = json.Unmarshal(data.Bytes(), task)
	if err != nil {
		return err
	}

	output, err := task.Apply()
	if err != nil {
		return err
	}

	result, err := json.Marshal(output)
	if err != nil {
		return err
	}

	_, err = fmt.Printf("%s\n", stringx.Sanitize(result))
	return err
}
