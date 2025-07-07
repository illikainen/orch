//lint:ignore ST1003 readability
package qvm_firewall // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
)

func init() {
	fn.Must(worker.Register("qvm_firewall", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	status, changes, err := e.Firewall.Apply(e.Name, e.Config.DryRun)
	if err != nil {
		return nil, err
	}

	return &outputs.Output{
		Status: status,
		Diff: map[string][]string{
			"firewall": changes,
		},
	}, nil
}
