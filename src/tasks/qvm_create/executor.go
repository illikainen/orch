//lint:ignore ST1003 readability
package qvm_create // revive:disable-line:var-naming

import (
	"fmt"

	"github.com/illikainen/orch/src/qubes"
	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"
	"github.com/illikainen/orch/src/tasks/qvm_prefs"
	"github.com/illikainen/orch/src/tasks/qvm_service"

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
	var changes []string
	status := outputs.StatusUnchanged

	exists, err := qubes.Check(e.Name)
	if err != nil {
		return nil, err
	}
	if !exists {
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

		changes = append(changes, fmt.Sprintf("%s: created", e.Name))
		status = outputs.StatusChanged
	}

	var prefChanges []string
	var svcChanges []string
	var fwChanges []string
	var featChanges []string

	if exists || !e.Config.DryRun {
		prefStatus, tmp, err := qvm_prefs.ApplyPreferences(e.Name, &e.Preferences, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		if status == outputs.StatusUnchanged {
			status = prefStatus
		}
		prefChanges = tmp

		svcStatus, tmp, err := qvm_service.Apply(e.Name, &e.Services, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		if status == outputs.StatusUnchanged {
			status = svcStatus
		}
		svcChanges = tmp

		fwStatus, tmp, err := e.Firewall.Apply(e.Name, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		if fwStatus == outputs.StatusUnchanged {
			status = fwStatus
		}
		fwChanges = tmp

		featStatus, tmp, err := e.Features.Apply(e.Name, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		if featStatus == outputs.StatusUnchanged {
			status = featStatus
		}
		featChanges = tmp
	}

	return &outputs.Output{
		Status: status,
		Diff: map[string][]string{
			"vm":          changes,
			"preferences": prefChanges,
			"services":    svcChanges,
			"firewall":    fwChanges,
			"features":    featChanges,
		},
	}, nil
}
