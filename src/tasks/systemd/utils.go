package systemd

import (
	"fmt"
	"strings"

	"github.com/illikainen/go-utils/src/process"
)

func Start(name string, dryRun bool) ([]string, error) {
	p, err := process.Exec(&process.ExecOptions{
		Command:         []string{"systemctl", "is-active", "--quiet", "--", name},
		IgnoreExitError: true,
	})
	if err != nil {
		return nil, err
	}

	if p.ExitCode != 0 {
		if !dryRun {
			_, err := process.Exec(&process.ExecOptions{
				Command: []string{"systemctl", "start", "--", name},
			})
			if err != nil {
				return nil, err
			}
		}

		return []string{fmt.Sprintf("%s: started", name)}, nil
	}

	return nil, nil
}

func Stop(name string, dryRun bool) ([]string, error) {
	p, err := process.Exec(&process.ExecOptions{
		Command:         []string{"systemctl", "is-active", "--quiet", "--", name},
		IgnoreExitError: true,
	})
	if err != nil {
		return nil, err
	}

	if p.ExitCode == 0 {
		if !dryRun {
			_, err := process.Exec(&process.ExecOptions{
				Command: []string{"systemctl", "stop", "--", name},
			})
			if err != nil {
				return nil, err
			}
		}

		return []string{fmt.Sprintf("%s: stopped", name)}, nil
	}

	return nil, nil
}

func Restart(name string, dryRun bool) ([]string, error) {
	p, err := process.Exec(&process.ExecOptions{
		Command:         []string{"systemctl", "is-active", "--quiet", "--", name},
		IgnoreExitError: true,
	})
	if err != nil {
		return nil, err
	}

	if p.ExitCode == 0 {
		if !dryRun {
			_, err := process.Exec(&process.ExecOptions{
				Command: []string{"systemctl", "restart", "--", name},
			})
			if err != nil {
				return nil, err
			}
		}

		return []string{fmt.Sprintf("%s: restarted", name)}, nil
	}

	return nil, nil
}

func Enable(name string, dryRun bool) ([]string, error) {
	p, err := process.Exec(&process.ExecOptions{
		Command:         []string{"systemctl", "is-enabled", "--quiet", "--", name},
		IgnoreExitError: true,
	})
	if err != nil {
		return nil, err
	}

	if p.ExitCode != 0 {
		if !dryRun {
			_, err := process.Exec(&process.ExecOptions{
				Command: []string{"systemctl", "enable", "--", name},
			})
			if err != nil {
				return nil, err
			}
		}

		return []string{fmt.Sprintf("%s: enabled", name)}, nil
	}

	return nil, nil
}

func Disable(name string, dryRun bool) ([]string, error) {
	changes, err := Stop(name, dryRun)
	if err != nil {
		return nil, err
	}

	p, err := process.Exec(&process.ExecOptions{
		Command:         []string{"systemctl", "is-enabled", "--quiet", "--", name},
		IgnoreExitError: true,
	})
	if err != nil {
		return nil, err
	}

	if p.ExitCode == 0 {
		if !dryRun {
			_, err := process.Exec(&process.ExecOptions{
				Command: []string{"systemctl", "disable", "--", name},
			})
			if err != nil {
				return nil, err
			}
		}

		changes = append(changes, fmt.Sprintf("%s: disabled", name))
	}

	return changes, nil
}

func Mask(name string, dryRun bool) ([]string, error) {
	changes, err := Stop(name, dryRun)
	if err != nil {
		return nil, err
	}

	p, err := process.Exec(&process.ExecOptions{
		Command:         []string{"systemctl", "is-enabled", "--", name},
		IgnoreExitError: true,
	})
	if err != nil {
		return nil, err
	}

	if strings.TrimRight(string(p.Stdout), " \n") != "masked" {
		if !dryRun {
			_, err := process.Exec(&process.ExecOptions{
				Command: []string{"systemctl", "mask", "--", name},
			})
			if err != nil {
				return nil, err
			}
		}

		changes = append(changes, fmt.Sprintf("%s: masked", name))
	}

	return changes, nil
}

func Unmask(name string, dryRun bool) ([]string, error) {
	p, err := process.Exec(&process.ExecOptions{
		Command:         []string{"systemctl", "is-enabled", "--", name},
		IgnoreExitError: true,
	})
	if err != nil {
		return nil, err
	}

	if p.ExitCode != 0 && strings.TrimRight(string(p.Stdout), " \n") == "masked" {
		if !dryRun {
			_, err := process.Exec(&process.ExecOptions{
				Command: []string{"systemctl", "unmask", "--", name},
			})
			if err != nil {
				return nil, err
			}
		}

		return []string{fmt.Sprintf("%s: unmasked", name)}, nil
	}

	return nil, nil
}
