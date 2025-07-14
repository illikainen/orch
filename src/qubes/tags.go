package qubes

import (
	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/process"
	"github.com/illikainen/go-utils/src/stringx"
	"golang.org/x/exp/slices"

	"github.com/illikainen/orch/src/tasks/outputs"
)

type Tags []string

func (t *Tags) Apply(name string, dryRun bool) (int, []string, error) {
	if dryRun {
		exists, err := Check(name)
		if err != nil {
			return 0, nil, err
		}

		if !exists {
			return outputs.StatusIndeterminate, nil, nil
		}
	}

	if len(*t) == 0 {
		return outputs.StatusUnchanged, nil, nil
	}

	p, err := process.Exec(&process.ExecOptions{
		Command: []string{"qvm-tags", "--", name},
	})

	curTags := stringx.SplitLines(string(p.Stdout))
	slices.Sort(curTags)
	slices.Sort(*t)

	var changes []string
	if !slices.Equal(*t, curTags) {
		for _, tag := range curTags {
			if !slices.Contains(*t, tag) {
				if !dryRun {
					_, err := process.Exec(&process.ExecOptions{
						Command: []string{"qvm-tags", "--", name, "del", tag},
					})
					if err != nil {
						return 0, nil, err
					}
				}
				changes = append(changes, "-"+tag)
			}
		}

		for _, tag := range *t {
			if !slices.Contains(curTags, tag) {
				if !dryRun {
					_, err := process.Exec(&process.ExecOptions{
						Command: []string{"qvm-tags", "--", name, "add", tag},
					})
					if err != nil {
						return 0, nil, err
					}
				}
				changes = append(changes, "+"+tag)
			}
		}
	}

	return fn.Ternary(changes == nil, outputs.StatusUnchanged, outputs.StatusChanged), changes, err
}
