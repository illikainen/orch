//lint:ignore ST1003 readability
package file_remove // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
)

func init() {
	fn.Must(worker.Register("file_remove", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	changes, err := Remove(e.Path, e.Config.DryRun)
	if err != nil {
		return nil, err
	}

	return &outputs.Output{
		Changed: changes != nil,
		Diff: map[string][]string{
			"remove": changes,
		},
	}, nil
}
