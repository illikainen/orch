package group

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"

	"github.com/pkg/errors"

	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/process"
	"github.com/illikainen/go-utils/src/seq"
)

func init() {
	fn.Must(worker.Register("group", NewExecutor))
}

type Executor struct {
	Tasks  []Task          `json:"tasks"`
	Config *configs.Config `json:"config" hcl:"-"`
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (output any, err error) {
	var changes []string

	for i := range e.Tasks {
		if e.Tasks[i].State == "present" {
			g, err := user.LookupGroup(e.Tasks[i].Name)
			if err != nil {
				if _, ok := err.(user.UnknownGroupError); !ok {
					return nil, errors.WithStack(err)
				}

				if err := groupAdd(&e.Tasks[i], e.Config.DryRun, &changes); err != nil {
					return nil, err
				}
			} else {
				if err := ensureUsers(g, e.Tasks[i].Users, e.Config.DryRun, &changes); err != nil {
					return nil, err
				}
			}
		}
	}

	return &outputs.Output{
		Status: fn.Ternary(changes == nil, outputs.StatusUnchanged, outputs.StatusChanged),
		Diff: map[string][]string{
			"group": changes,
		},
	}, nil
}

func groupAdd(task *Task, dryRun bool, changes *[]string) error {
	if groupadd, err := exec.LookPath("groupadd"); err == nil { // revive:disable-line
		cmd := []string{groupadd}
		if task.System {
			cmd = append(cmd, "--system")
		}
		if task.Users != nil {
			cmd = append(cmd, "--users", strings.Join(task.Users, ","))
		}
		cmd = append(cmd, task.Name)

		if !dryRun {
			_, err := process.Exec(&process.ExecOptions{
				Command: cmd,
			})
			if err != nil {
				return err
			}
		}
	} else {
		return errors.Errorf("%s: can't add group", task.Name)
	}

	*changes = append(*changes, fmt.Sprintf("%s: created", task.Name))
	return nil
}

func ensureUsers(grp *user.Group, users []string, dryRun bool, changes *[]string) error {
	for _, u := range users {
		usr, err := user.Lookup(u)
		if err != nil {
			return errors.WithStack(err)
		}

		groupIds, err := usr.GroupIds()
		if err != nil {
			return errors.WithStack(err)
		}

		if usr.Gid != grp.Gid && !seq.Contains(groupIds, grp.Gid) {
			if usermod, err := exec.LookPath("usermod"); err == nil { // revive:disable-line
				if !dryRun {
					_, err := process.Exec(&process.ExecOptions{
						Command: []string{usermod, "--append", "--groups", grp.Name, u},
					})
					if err != nil {
						return err
					}
				}
			} else {
				return errors.Errorf("%s: can't add '%s'", grp.Name, u)
			}

			*changes = append(*changes, fmt.Sprintf("%s: added %s", grp.Name, u))
		}
	}

	return nil
}
