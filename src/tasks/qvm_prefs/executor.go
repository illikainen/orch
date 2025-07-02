//lint:ignore ST1003 readability
package qvm_prefs // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
)

func init() {
	fn.Must(worker.Register("qvm_prefs", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	status, changes, err := ApplyPreferences(e.Name, &e.Preferences, e.Config.DryRun)
	if err != nil {
		return nil, err
	}

	return &outputs.Output{
		Status: status,
		Diff: map[string][]string{
			"preferences": changes,
		},
	}, nil
}
