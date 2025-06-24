package qubes

import (
	"io"

	"github.com/illikainen/orch/src/fact"

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

var facts *fact.OS

func Command(opts *CommandOptions) ([]string, error) {
	if facts == nil {
		tmp, err := fact.GatherOSFacts()
		if err != nil {
			return nil, err
		}
		facts = tmp
	}

	var args []string
	if facts.Name == "qubes" {
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
