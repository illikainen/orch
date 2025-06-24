//lint:ignore ST1003 readability
package dir_manage // revive:disable-line:var-naming

import (
	"encoding/base64"
	"path/filepath"

	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/file_manage"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
)

func init() {
	fn.Must(worker.Register("dir_manage", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	var dirChanges []string
	var fileChanges []string
	var permChanges []string

	for path, content := range e.Content {
		data, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, err
		}

		newDirChanges, err := file_manage.Mkdir(filepath.Dir(path), e.DirMode, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		dirChanges = append(dirChanges, newDirChanges...)

		newFileChanges, err := file_manage.WriteFile(path, data, e.FileMode, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		fileChanges = append(fileChanges, newFileChanges...)

		newPermChanges, err := file_manage.Chmod(path, e.FileMode, e.Config.DryRun)
		if err != nil {
			return nil, err
		}
		permChanges = append(permChanges, newPermChanges...)
	}

	return &outputs.Output{
		Changed: dirChanges != nil || fileChanges != nil || permChanges != nil,
		Diff: map[string][]string{
			"mkdir":       dirChanges,
			"file":        fileChanges,
			"permissions": permChanges,
		},
	}, nil
}
