package patch

import (
	"encoding/base64"

	"github.com/pkg/errors"

	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
)

func init() {
	fn.Must(worker.Register("patch", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	patch, err := base64.StdEncoding.DecodeString(e.Content)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	changes, err := Patch(e.Dir, patch, e.Strip, e.Config.DryRun)
	if err != nil {
		return nil, err
	}

	return &outputs.Output{
		Changed: changes != nil,
		Diff: map[string][]string{
			"patch": changes,
		},
	}, nil
}
