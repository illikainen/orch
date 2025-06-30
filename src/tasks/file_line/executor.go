//lint:ignore ST1003 readability
package file_line // revive:disable-line:var-naming

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/iofs"
)

func init() {
	fn.Must(worker.Register("file_line", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	data, err := iofs.ReadFile(e.Path)
	if err != nil {
		return nil, err
	}

	var changes []string
	var lines []string

	match := false
	scan := bufio.NewScanner(bytes.NewReader(data))
	for scan.Scan() {
		line := scan.Text()
		if e.Regexp.MatchString(line) {
			if line != e.Line {
				change := fmt.Sprintf("%s: replace '%s' with '%s'", e.Path, line, e.Line)
				changes = append(changes, change)
			}
			match = true
			lines = append(lines, e.Line)
		} else {
			lines = append(lines, line)
		}
	}

	err = scan.Err()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if !match {
		changes = append(changes, fmt.Sprintf("%s: add '%s'", e.Path, e.Line))
		lines = append(lines, e.Line)
	}

	if changes != nil && !e.Config.DryRun {
		f, err := os.OpenFile(e.Path, os.O_WRONLY, 0)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		nl := fn.Ternary(runtime.GOOS == "windows", "\r\n", "\n")
		err = iofs.Copy(f, strings.NewReader(strings.Join(lines, nl)+nl))
		if err != nil {
			return nil, err
		}
	}

	return &outputs.Output{
		Status: fn.Ternary(changes == nil, outputs.StatusUnchanged, outputs.StatusChanged),
		Diff: map[string][]string{
			"line": changes,
		},
	}, nil
}
