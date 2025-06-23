//lint:ignore ST1003 readability
package file_manage // revive:disable-line:var-naming

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/orch/src/utils"

	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/stringx"
	"github.com/pkg/errors"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func Mkdir(name string, mode os.FileMode, dryRun bool) ([]string, error) {
	var changes []string
	path := ""

	for i, part := range strings.Split(name, string(filepath.Separator)) {
		if i == 0 && part == "" {
			part = string(filepath.Separator)
		}
		path = filepath.Join(path, part)

		exists, err := iofs.Exists(path)
		if err != nil {
			return nil, err
		}

		if !exists {
			if !dryRun {
				err := os.Mkdir(path, mode)
				if err != nil {
					return nil, err
				}
			}
			changes = append(changes, fmt.Sprintf("%s: %s (%#o)", path, mode, mode))
		}
	}

	return changes, nil
}

func Chmod(name string, mode os.FileMode, dryRun bool) ([]string, error) {
	stat, err := os.Stat(name)
	if err != nil {
		if dryRun && errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	oldMode := stat.Mode().Perm()
	if oldMode != mode {
		if !dryRun {
			err := os.Chmod(name, mode)
			if err != nil {
				return nil, err
			}
		}

		return []string{
			fmt.Sprintf("%s: %s (%#o) -> %s (%#o)", name, oldMode, int(oldMode), mode, int(mode)),
		}, nil
	}

	return nil, nil
}

func WriteFile(name string, data []byte, mode os.FileMode, dryRun bool) ([]string, error) {
	cur, err := iofs.ReadFile(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !dryRun {
				err := os.WriteFile(name, data, mode)
				if err != nil {
					return nil, err
				}
			}
			return []string{fmt.Sprintf("%s: wrote %d bytes", name, len(data))}, nil
		}

		return nil, err
	}

	if !bytes.Equal(cur, data) {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(cur), string(data), true)
		diff, err := utils.FormatDiff(diffs)
		if err != nil {
			return nil, err
		}

		if !dryRun {
			err := os.WriteFile(name, data, mode)
			if err != nil {
				return nil, err
			}
		}

		return stringx.SplitLines(diff), nil
	}

	return nil, nil
}
