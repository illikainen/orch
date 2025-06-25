//lint:ignore ST1003 readability
package systemd_daemon_reload // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/process"
)

func init() {
	fn.Must(worker.Register("systemd_daemon_reload", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	if !e.Config.DryRun {
		_, err := process.Exec(&process.ExecOptions{
			Command: []string{"systemctl", "daemon-reload"},
		})
		if err != nil {
			return nil, err
		}
	}

	return &outputs.Output{
		Changed: true,
	}, nil
}
