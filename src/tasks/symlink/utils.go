package symlink

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/illikainen/orch/src/tasks/file_remove"
)

func Symlink(src string, dst string, dryRun bool) ([]string, error) {
	var changes []string

	stat, err := os.Lstat(dst)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, errors.WithStack(err)
		}
	} else {
		if stat.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(dst)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			if target == src {
				return nil, nil
			}
		}

		rmChanges, err := file_remove.Remove(dst, dryRun)
		if err != nil {
			return nil, err
		}
		changes = append(changes, rmChanges...)
	}

	if !dryRun {
		err := os.Symlink(src, dst)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return append(changes, fmt.Sprintf("%s: symlink from %s", dst, src)), nil
}
