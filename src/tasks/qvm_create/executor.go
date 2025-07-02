//lint:ignore ST1003 readability
package qvm_create // revive:disable-line:var-naming

import (
	"fmt"

	"github.com/illikainen/orch/src/qubes"
	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"
	"github.com/illikainen/orch/src/tasks/qvm_prefs"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/process"
)

func init() {
	fn.Must(worker.Register("qvm_create", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	exists, err := qubes.Check(e.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		status, changes, err := qvm_prefs.ApplyPreferences(e.Name, &e.Preferences, e.Config.DryRun)
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

	if !e.Config.DryRun {
		cmd := []string{"qvm-create"}

		if e.Preferences.Label != nil {
			cmd = append(cmd, "--label", *e.Preferences.Label)
		}

		if e.Preferences.Class != nil {
			cmd = append(cmd, "--class", *e.Preferences.Class)
		}

		if e.Preferences.Template != nil {
			cmd = append(cmd, "--template", *e.Preferences.Template)
		}

		if e.Preferences.NetVM != nil {
			cmd = append(cmd, fmt.Sprintf("--property=netvm=%s", *e.Preferences.NetVM))
		}

		cmd = append(cmd, "--", e.Name)

		_, err := process.Exec(&process.ExecOptions{
			Command: cmd,
		})
		if err != nil {
			return nil, err
		}
	}

	_, changes, err := qvm_prefs.ApplyPreferences(e.Name, &e.Preferences, e.Config.DryRun)
	if err != nil {
		return nil, err
	}

	return &outputs.Output{
		Status: outputs.StatusChanged,
		Diff: map[string][]string{
			"vm":          []string{fmt.Sprintf("%s: created", e.Name)},
			"preferences": changes,
		},
	}, nil
}
