package user

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
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/process"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/illikainen/go-utils/src/stringx"
)

func init() {
	fn.Must(worker.Register("user", NewExecutor))
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
			u, err := user.Lookup(e.Tasks[i].Name)
			if err != nil {
				if _, ok := err.(user.UnknownUserError); !ok {
					return nil, errors.WithStack(err)
				}

				if err := userAdd(&e.Tasks[i], e.Config.DryRun, &changes); err != nil {
					return nil, err
				}
			} else {
				if err := ensureGroup(u, e.Tasks[i].Group, e.Config.DryRun, &changes); err != nil {
					return nil, err
				}

				if err := ensureGroups(u, e.Tasks[i].Groups, e.Config.DryRun, &changes); err != nil {
					return nil, err
				}

				if err := ensureShell(u, e.Tasks[i].Shell, e.Config.DryRun, &changes); err != nil {
					return nil, err
				}
			}
		}
	}

	return &outputs.Output{
		Status: fn.Ternary(changes == nil, outputs.StatusUnchanged, outputs.StatusChanged),
		Diff: map[string][]string{
			"user": changes,
		},
	}, nil
}

func userAdd(task *Task, dryRun bool, changes *[]string) error {
	if useradd, err := exec.LookPath("useradd"); err == nil { // revive:disable-line
		cmd := []string{useradd}

		if task.Group != "" {
			cmd = append(cmd, "--gid", task.Group)
		} else {
			cmd = append(cmd, "--user-group")
		}

		if task.Groups != nil {
			cmd = append(cmd, "--groups", strings.Join(task.Groups, ","))
		}

		if task.HomeDir != "" {
			cmd = append(cmd, "--home-dir", task.HomeDir)
		}

		if task.CreateHome {
			cmd = append(cmd, "--create-home")
		} else {
			cmd = append(cmd, "--no-create-home")
		}

		if task.Shell != "" {
			cmd = append(cmd, "--shell", task.Shell)
		}

		if task.System {
			cmd = append(cmd, "--system")
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
		return errors.Errorf("%s: can't add user", task.Name)
	}

	*changes = append(*changes, fmt.Sprintf("%s: created", task.Name))
	return nil
}

func ensureGroup(usr *user.User, group string, dryRun bool, changes *[]string) error {
	if group == "" {
		return nil
	}

	grp, err := user.LookupGroup(group)
	if err != nil {
		return errors.WithStack(err)
	}

	if usr.Gid != grp.Gid {
		if usermod, err := exec.LookPath("usermod"); err == nil { // revive:disable-line
			if !dryRun {
				_, err := process.Exec(&process.ExecOptions{
					Command: []string{usermod, "--gid", group},
				})
				if err != nil {
					return err
				}
			}
		} else {
			return errors.Errorf("%s: can't add to group '%s'", usr.Username, group)
		}

		*changes = append(*changes, fmt.Sprintf("%s: added to '%s'", usr.Username, group))
	}
	return nil
}

func ensureGroups(usr *user.User, groups []string, dryRun bool, changes *[]string) error {
	for _, group := range groups {
		grp, err := user.LookupGroup(group)
		if err != nil {
			return errors.Wrap(err, group)
		}

		groupIds, err := usr.GroupIds()
		if err != nil {
			return errors.WithStack(err)
		}

		if !seq.Contains(groupIds, grp.Gid) {
			if usermod, err := exec.LookPath("usermod"); err == nil { // revive:disable-line
				if !dryRun {
					_, err := process.Exec(&process.ExecOptions{
						Command: []string{usermod, "--append", "--groups", group},
					})
					if err != nil {
						return err
					}
				}
			} else {
				return errors.Errorf("%s: can't add to group '%s'", usr.Username, group)
			}

			*changes = append(*changes, fmt.Sprintf("%s: added to '%s'", usr.Username, group))
		}
	}
	return nil
}

func ensureShell(usr *user.User, shell string, dryRun bool, changes *[]string) error {
	if shell == "" {
		return nil
	}

	data, err := iofs.ReadFile("/etc/passwd")
	if err != nil {
		return err
	}

	for _, line := range stringx.SplitLines(string(data)) {
		elts := strings.Split(line, ":")
		if len(elts) >= 7 && elts[0] == usr.Username && elts[2] == usr.Gid {
			curShell := elts[6]
			if shell != curShell {
				if usermod, err := exec.LookPath("usermod"); err == nil { // revive:disable-line
					if !dryRun {
						_, err := process.Exec(&process.ExecOptions{
							Command: []string{usermod, "--shell", shell, usr.Username},
						})
						if err != nil {
							return err
						}
					}
				} else {
					return errors.Errorf("%s: can't change shell to %s", usr.Username, shell)
				}

				*changes = append(*changes, fmt.Sprintf("%s: changed shell from '%s' to '%s'",
					usr.Username, curShell, shell))
			}
		}
	}

	return nil
}
