package qubes

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/illikainen/go-utils/src/process"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
)

type VM struct {
	Name     string
	Class    string
	State    int
	Label    string
	NetVM    string
	Template string
}

const (
	HaltedState = iota
	RunningState
	PausedState
	TransientState
)

func List() ([]*VM, error) {
	out, err := process.Exec(&process.ExecOptions{
		Command: []string{
			"qvm-ls",
			"--raw-data",
			"--fields",
			"name,class,state,label,netvm,template",
		},
	})
	if err != nil {
		return nil, err
	}

	var vms []*VM
	scan := bufio.NewScanner(bytes.NewReader(out.Stdout))
	for scan.Scan() {
		line := scan.Text()
		parts := strings.Split(line, "|")
		if len(parts) != 6 {
			return nil, errors.Errorf("invalid line: %s", line)
		}

		var state int
		switch parts[2] {
		case "Halted":
			state = HaltedState
		case "Running":
			state = RunningState
		case "Paused":
			state = PausedState
		case "Transient":
			state = TransientState
		default:
			return nil, errors.Errorf("invalid vm state: %s", parts[2])
		}

		vms = append(vms, &VM{
			Name:     parts[0],
			Class:    parts[1],
			State:    state,
			Label:    parts[3],
			NetVM:    parts[4],
			Template: parts[5],
		})
	}

	err = scan.Err()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vms, nil
}

func Find(name string) (*VM, error) {
	vms, err := List()
	if err != nil {
		return nil, err
	}

	vm, found := seq.FindBy(vms, func(elt *VM) bool {
		return elt.Name == name
	})
	if !found {
		return nil, errors.Errorf("%s does not exist", name)
	}

	return vm, nil
}
