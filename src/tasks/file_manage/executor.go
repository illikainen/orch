//lint:ignore ST1003 readability
package file_manage // revive:disable-line:var-naming

import (
	"encoding/base64"
	"path/filepath"

	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
)

func init() {
	fn.Must(worker.Register("file_manage", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	srcData, err := base64.StdEncoding.DecodeString(e.Content)
	if err != nil {
		return nil, err
	}

	dirChanges, err := Mkdir(filepath.Dir(e.Dst), e.DirMode, e.Config.DryRun)
	if err != nil {
		return nil, err
	}

	var permDirChanges []string
	if !e.IgnoreDirMode {
		permDirChanges, err = Chmod(filepath.Dir(e.Dst), e.DirMode, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
	}

	fileChanges, err := WriteFile(e.Dst, srcData, e.FileMode, e.Config.DryRun)
	if err != nil {
		return nil, err
	}

	permFileChanges, err := Chmod(e.Dst, e.FileMode, e.Config.DryRun)
	if err != nil {
		return nil, err
	}

	return &outputs.Output{
		Changed: dirChanges != nil || fileChanges != nil || permDirChanges != nil || permFileChanges != nil,
		Diff: map[string][]string{
			"mkdir":       dirChanges,
			"file":        fileChanges,
			"permissions": append(permDirChanges, permFileChanges...),
		},
	}, nil
}
