package systemd

import (
	"github.com/pkg/errors"

	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
)

func init() {
	fn.Must(worker.Register("systemd", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	actions := map[string]func(string, bool) ([]string, error){
		"start":   Start,
		"stop":    Stop,
		"restart": Restart,
		"enable":  Enable,
		"disable": Disable,
		"mask":    Mask,
		"unmask":  Unmask,
	}

	fun, ok := actions[e.Action]
	if !ok {
		return nil, errors.Errorf("invalid action: %s", e.Action)
	}

	changes, err := fun(e.Name, e.Config.DryRun)
	if err != nil {
		return nil, err
	}

	return &outputs.Output{
		Changed: changes != nil,
		Diff: map[string][]string{
			"changes": changes,
		},
	}, nil
}
