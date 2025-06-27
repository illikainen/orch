package symlink

import (
	"os"
	"path/filepath"

	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/file_manage"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
)

func init() {
	fn.Must(worker.Register("symlink", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	var changes []string

	src := e.Src
	if !filepath.IsAbs(src) {
		src = filepath.Join(e.BaseDir, src)
	}

	if e.LinkContents {
		elts, err := os.ReadDir(src)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		for _, elt := range elts {
			exclude := seq.ContainsBy(e.Exclude, func(exclude string) bool {
				match, e := filepath.Match(exclude, elt.Name())
				if e != nil {
					err = e
					return false
				}
				return match
			})
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if !exclude {
				mkdirChanges, err := file_manage.Mkdir(e.Dst, e.DirMode, e.Config.DryRun)
				if err != nil {
					return nil, err
				}
				changes = append(changes, mkdirChanges...)

				symlinkChanges, err := Symlink(
					filepath.Join(src, elt.Name()),
					filepath.Join(e.Dst, elt.Name()),
					e.Config.DryRun,
				)
				if err != nil {
					return nil, err
				}
				changes = append(changes, symlinkChanges...)
			}
		}
	} else {
		mkdirChanges, err := file_manage.Mkdir(filepath.Dir(e.Dst), e.DirMode, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		changes = append(changes, mkdirChanges...)

		symlinkChanges, err := Symlink(src, e.Dst, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		changes = append(changes, symlinkChanges...)
	}

	return &outputs.Output{
		Changed: changes != nil,
		Diff: map[string][]string{
			"changes": changes,
		},
	}, nil
}
