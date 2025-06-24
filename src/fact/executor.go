package fact

import (
	"os"

	"github.com/illikainen/orch/src/qubes"
	"github.com/illikainen/orch/src/rpc/worker"

	"github.com/illikainen/go-utils/src/fn"
)

func init() {
	fn.Must(worker.Register("gather_facts", NewExecutor))
}

type Executor struct {
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	facts := &Facts{}

	var err error
	facts.Hostname, err = os.Hostname()
	if err != nil {
		return nil, err
	}

	facts.OS, err = GatherOSFacts()
	if err != nil {
		return nil, err
	}

	facts.IsQVM, err = qubes.IsQVM()
	if err != nil {
		return nil, err
	}

	return facts, nil
}
