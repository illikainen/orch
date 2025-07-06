//lint:ignore ST1003 readability
package qvm_service // revive:disable-line:var-naming

import (
	"github.com/illikainen/go-utils/src/fn"

	"github.com/illikainen/orch/src/qubes"
	"github.com/illikainen/orch/src/tasks/outputs"
)

func Apply(name string, services *qubes.Services, dryRun bool) (int, []string, error) {
	if dryRun {
		exists, err := qubes.Check(name)
		if err != nil {
			return 0, nil, err
		}

		if !exists {
			return outputs.StatusIndeterminate, nil, nil
		}
	}

	changes, err := services.Apply(name, dryRun)
	if err != nil {
		return 0, nil, err
	}

	return fn.Ternary(changes == nil, outputs.StatusUnchanged, outputs.StatusChanged), changes, nil
}
