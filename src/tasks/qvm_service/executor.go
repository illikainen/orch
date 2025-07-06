//lint:ignore ST1003 readability
package qvm_service // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
)

func init() {
	fn.Must(worker.Register("qvm_service", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	status, changes, err := Apply(e.Name, &e.Services, e.Config.DryRun)
	if err != nil {
		return nil, err
	}

	return &outputs.Output{
		Status: status,
		Diff: map[string][]string{
			"services": changes,
		},
	}, nil
}
