//lint:ignore ST1003 readability
package file_remove // revive:disable-line:var-naming

import (
	"fmt"
	"os"

	"github.com/illikainen/go-utils/src/iofs"
	"github.com/pkg/errors"
)

func Remove(name string, dryRun bool) ([]string, error) {
	_, err := os.Stat(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}

	if !dryRun {
		err := iofs.Remove(name)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return []string{
		fmt.Sprintf("%s: removed", name),
	}, nil
}
