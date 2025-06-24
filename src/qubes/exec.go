package qubes

import (
	"io"

	"github.com/illikainen/go-utils/src/process"
)

type ExecOptions struct {
	Name    string
	Command []string
	Become  string
	Stdin   io.Reader
	Stdout  process.OutputFunc
	Stderr  process.OutputFunc
}

func Exec(opts *ExecOptions) (*process.ExecOutput, error) {
	args, err := Command(&CommandOptions{
		Name:    opts.Name,
		Command: opts.Command,
		Become:  opts.Become,
	})
	if err != nil {
		return nil, err
	}

	o := &process.ExecOptions{
		Command: args,
		Stdin:   opts.Stdin,
		Stdout:  opts.Stdout,
		Stderr:  opts.Stderr,
	}
	return process.Exec(o)
}

type CommandOptions struct {
	Name    string
	Command []string
	Become  string
}

func Command(opts *CommandOptions) ([]string, error) {
	dom0, err := IsDom0()
	if err != nil {
		return nil, err
	}

	var args []string
	if dom0 {
		args = []string{
			"/usr/bin/qvm-run",
			"--no-autostart",
			"--pass-io",
			"--no-gui",
			"--no-color-output",
			"--no-color-stderr",
			"--filter-escape-chars",
			"--no-shell",
		}

		if opts.Become != "" {
			args = append(args, "--user", opts.Become)
		}

		args = append(args, "--", opts.Name)
	} else {
		args = []string{
			"/bin/sh",
			"/usr/bin/qvm-run-vm",
			"--",
			opts.Name,
		}

		if opts.Become != "" {
			args = append(args, become(opts.Become)...)
		}
	}

	return append(args, opts.Command...), nil
}

func become(username string) []string {
	return []string{"sudo", "-u", username}
}
