package qvm

import (
	"fmt"

	"github.com/illikainen/orch/src/qubes"
	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/process"
)

func init() {
	fn.Must(worker.Register("qvm", NewExecutor))
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
			var cmd []string
			if e.Clone != nil {
				cmd = []string{"qvm-clone"}
			} else {
				cmd = []string{"qvm-create"}

				if e.Preferences.Label != nil {
					cmd = append(cmd, "--label", *e.Preferences.Label)
				}

				if e.Preferences.Template != nil {
					cmd = append(cmd, "--template", *e.Preferences.Template)
				}

				if e.Preferences.NetVM != nil {
					cmd = append(cmd, fmt.Sprintf("--property=netvm=%s", *e.Preferences.NetVM))
				}
			}

			if e.Preferences.Class != nil {
				cmd = append(cmd, "--class", *e.Preferences.Class)
			}

			cmd = append(cmd, "--")
			if e.Clone != nil {
				cmd = append(cmd, *e.Clone)
			}
			cmd = append(cmd, e.Name)

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
	var tagChanges []string
	var volChanges []string

	if exists || !e.Config.DryRun {
		prefStatus, tmp, err := e.Preferences.Apply(e.Name, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		if status == outputs.StatusChanged {
			status = prefStatus
		}
		prefChanges = tmp

		svcStatus, tmp, err := e.Services.Apply(e.Name, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		if status == outputs.StatusChanged {
			status = svcStatus
		}
		svcChanges = tmp

		fwStatus, tmp, err := e.Firewall.Apply(e.Name, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		if fwStatus == outputs.StatusChanged {
			status = fwStatus
		}
		fwChanges = tmp

		featStatus, tmp, err := e.Features.Apply(e.Name, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		if featStatus == outputs.StatusChanged {
			status = featStatus
		}
		featChanges = tmp

		tagStatus, tmp, err := e.Tags.Apply(e.Name, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		if tagStatus == outputs.StatusChanged {
			status = tagStatus
		}
		tagChanges = tmp

		for _, vol := range e.Volumes {
			volStatus, tmp, err := vol.Apply(e.Name, e.Config.DryRun)
			if err != nil {
				return nil, err
			}
			if volStatus == outputs.StatusChanged {
				status = volStatus
			}
			volChanges = append(volChanges, tmp...)
		}
	}

	return &outputs.Output{
		Status: status,
		Diff: map[string][]string{
			"vm":          changes,
			"preferences": prefChanges,
			"services":    svcChanges,
			"firewall":    fwChanges,
			"features":    featChanges,
			"tags":        tagChanges,
			"volumes":     volChanges,
		},
	}, nil
}
