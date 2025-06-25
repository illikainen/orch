package exec

import (
	"github.com/pkg/errors"

	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/process"
	"github.com/illikainen/go-utils/src/stringx"
	"github.com/kballard/go-shellquote"
)

func init() {
	fn.Must(worker.Register("exec", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	var cmd []string
	if e.Shell {
		cmd = []string{"/bin/sh", "-c", "--", e.Cmd}
	} else {
		s, err := shellquote.Split(e.Cmd)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		cmd = s
	}

	diff := map[string][]string{}
	if !e.Config.DryRun {
		out, err := process.Exec(&process.ExecOptions{
			Command: cmd,
		})
		if err != nil {
			return nil, err
		}

		if len(out.Stdout) != 0 {
			diff["stdout"] = stringx.SplitLines(string(out.Stdout))
		}
		if len(out.Stderr) != 0 {
			diff["stderr"] = stringx.SplitLines(string(out.Stderr))
		}
	}

	return &outputs.Output{
		Changed: true,
		Diff:    diff,
	}, nil
}
