package patch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/process"
	"github.com/pkg/errors"
)

func Patch(dir string, patch []byte, strip int, dryRun bool) (changes []string, err error) {
	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	tmp, tmpClean, err := iofs.MkdirTemp()
	if err != nil {
		return nil, err
	}
	defer errorx.Defer(tmpClean, &err)

	err = copyDir(dir, tmp)
	if err != nil {
		return nil, err
	}

	oldSum256, err := sha256Dir(tmp)
	if err != nil {
		return nil, err
	}

	changed, err := patchDir(tmp, patch, strip)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, nil
	}

	newSum256, err := sha256Dir(tmp)
	if err != nil {
		return nil, err
	}

	for src, cksum := range newSum256 {
		rel, err := filepath.Rel(tmp, src)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		dst := filepath.Join(dir, rel)

		oldCksum, ok := oldSum256[src]
		if !ok {
			if !dryRun {
				err := iofs.MoveFile(src, dst)
				if err != nil {
					return nil, err
				}
			}
			changes = append(changes, fmt.Sprintf("%s: new file", dst))
		} else if cksum != oldCksum {
			if !dryRun {
				err := iofs.MoveFile(src, dst)
				if err != nil {
					return nil, err
				}
			}
			changes = append(changes, fmt.Sprintf("%s: changed", dst))
		}
	}

	for src := range oldSum256 {
		rel, err := filepath.Rel(tmp, src)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		dst := filepath.Join(dir, rel)

		_, ok := newSum256[src]
		if !ok {
			if !dryRun {
				err := iofs.Remove(dst)
				if err != nil {
					return nil, err
				}
			}
			changes = append(changes, fmt.Sprintf("%s: removed", dst))
		}
	}

	return changes, nil
}

func patchDir(dir string, patch []byte, strip int) (bool, error) {
	busybox, err := exec.LookPath("busybox")
	if err == nil {
		p, err := process.Exec(&process.ExecOptions{
			Command:         []string{busybox, "patch", "-p", strconv.Itoa(strip)},
			Stdin:           bytes.NewReader(patch),
			Dir:             dir,
			IgnoreExitError: true,
		})
		if err != nil {
			return false, err
		}

		if p.ExitCode == 0 {
			return true, nil
		} else if p.ExitCode == 1 && strings.HasPrefix(string(p.Stderr), "Possibly reversed hunk") {
			return false, nil
		}

		return false, errors.Errorf("unknown error")
	}

	posix, err := exec.LookPath("patch")
	if err == nil {
		p, err := process.Exec(&process.ExecOptions{
			Command:         []string{posix, "-s", "-N", "-p", strconv.Itoa(strip)},
			Stdin:           bytes.NewReader(patch),
			Dir:             dir,
			IgnoreExitError: true,
		})
		if err != nil {
			return false, err
		}

		if p.ExitCode == 0 {
			return true, nil
		} else if p.ExitCode == 1 && strings.HasPrefix(string(p.Stdout),
			"Reversed (or previously applied) patch detected!  Skipping patch.") {
			return false, nil
		}

		return false, errors.Errorf("unknown error")
	}

	return false, errors.Errorf("missing patch program")
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(cur string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}

		if cur == src {
			return nil
		}

		rel, err := filepath.Rel(src, cur)
		if err != nil {
			return errors.WithStack(err)
		}

		curDst := filepath.Clean(filepath.Join(dst, rel))
		if !strings.HasPrefix(curDst, dst+string(os.PathSeparator)) {
			return errors.Errorf("invalid path: %s", curDst)
		}

		if d.IsDir() {
			return os.Mkdir(curDst, 0o700)
		}

		data, err := iofs.ReadFile(cur)
		if err != nil {
			return err
		}

		err = iofs.WriteFile(curDst, bytes.NewReader(data))
		if err != nil {
			return err
		}

		return nil
	})
}

func sha256Dir(src string) (map[string]string, error) {
	sum256 := map[string]string{}

	err := filepath.WalkDir(src, func(cur string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}

		if d.IsDir() {
			return nil
		}

		data, err := iofs.ReadFile(cur)
		if err != nil {
			return err
		}

		cksum := sha256.Sum256(data)
		sum256[cur] = hex.EncodeToString(cksum[:])

		return nil
	})
	if err != nil {
		return nil, err
	}

	return sum256, nil
}
