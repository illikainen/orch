package applytaskcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	rootcmd "github.com/illikainen/orch/src/cmd/root"
	"github.com/illikainen/orch/src/tasks"

	"github.com/illikainen/go-utils/src/logging"
	"github.com/illikainen/go-utils/src/stringx"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var command = &cobra.Command{
	Use:    "_apply-task",
	Short:  "Internal command to apply a task",
	RunE:   run,
	Hidden: true,
}

func Command(_ *rootcmd.Options) *cobra.Command {
	return command
}

func run(cmd *cobra.Command, _ []string) error {
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
