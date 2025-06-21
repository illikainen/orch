package qvm

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
	cmd := []string{"/bin/sh", "/usr/bin/qvm-run-vm", "--", opts.Name}
	if opts.Become != "" {
		cmd = append(cmd, become(opts.Become)...)
	}
	cmd = append(cmd, opts.Command...)

	o := &process.ExecOptions{
		Command: cmd,
		Stdin:   opts.Stdin,
		Stdout:  opts.Stdout,
		Stderr:  opts.Stderr,
	}
	return process.Exec(o)
}

// revive:disable-next-line:function-result-limit
func SandboxPaths() (ro []string, rw []string, dev []string, err error) {
	ro = []string{
		"/var/run/qubes/qrexec-agent",
	}

	dev = []string{
		"/dev/xen/evtchn",
		"/dev/xen/gntalloc",
		"/dev/xen/privcmd",
		"/dev/xen/xenbus",
	}

	return ro, nil, dev, nil
}

func become(username string) []string {
	return []string{"sudo", "-u", username}
}
