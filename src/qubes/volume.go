package qubes

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/process"
	"github.com/pkg/errors"

	"github.com/illikainen/orch/src/tasks/outputs"
)

type Volume struct {
	Name string `json:"name" hcl:"name"`
	Size string `json:"size" hcl:"size"`
}

func (v *Volume) Apply(name string, dryRun bool) (int, []string, error) {
	if dryRun {
		exists, err := Check(name)
		if err != nil {
			return 0, nil, err
		}

		if !exists {
			return outputs.StatusIndeterminate, nil, nil
		}
	}

	s := v.Size
	gb := false
	if strings.HasSuffix(s, "G") {
		s = strings.TrimSuffix(s, "G")
		gb = true
	}

	size, err := strconv.ParseInt(s, 10, reflect.TypeOf(int64(0)).Bits())
	if err != nil {
		return 0, nil, errors.WithStack(err)
	}

	// Input is trusted so the potential for overflows isn't much of a
	// concern.
	if gb {
		size *= 1000 * 1000 * 1000
	}

	p, err := process.Exec(&process.ExecOptions{
		Command: []string{"qvm-volume", "info", "--", fmt.Sprintf("%s:%s", name, v.Name), "size"},
	})
	if err != nil {
		return 0, nil, err
	}

	stdout := strings.TrimRight(string(p.Stdout), " \n")
	curSize, err := strconv.ParseInt(stdout, 10, reflect.TypeOf(int64(0)).Bits())
	if err != nil {
		return 0, nil, err
	}

	var changes []string
	if curSize < size {
		if !dryRun {
			_, err := process.Exec(&process.ExecOptions{
				Command: []string{"qvm-volume", "resize", "--", fmt.Sprintf("%s:%s", name, v.Name),
					fmt.Sprintf("%d", size)},
			})
			if err != nil {
				return 0, nil, err
			}
		}
		changes = append(changes, fmt.Sprintf("%s: %d -> %d", v.Name, curSize, size))
	}

	return fn.Ternary(changes == nil, outputs.StatusUnchanged, outputs.StatusChanged), changes, nil
}
