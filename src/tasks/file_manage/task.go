//lint:ignore ST1003 readability
package file_manage // revive:disable-line:var-naming

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/stringx"
	"github.com/pkg/errors"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition     bool            `json:"condition"`
	Src           string          `json:"src"`
	Dst           string          `json:"dst"`
	Content       string          `json:"content"`
	FileMode      os.FileMode     `json:"file_mode"`
	DirMode       os.FileMode     `json:"dir_mode"`
	IgnoreDirMode bool            `json:"ignore_dir_mode"`
	Config        *configs.Config `json:"config"`
	value         cty.Value
}

func (t *Task) Decode(body hcl.Body, ctx *hcl.EvalContext, config *configs.Config) error {
	value, diags := hcldec.Decode(
		body,
		&hcldec.ObjectSpec{
			"condition": &hcldec.AttrSpec{
				Name: "condition",
				Type: cty.Bool,
			},
			"src": &hcldec.AttrSpec{
				Name: "src",
				Type: cty.String,
			},
			"dst": &hcldec.AttrSpec{
				Name:     "dst",
				Type:     cty.String,
				Required: true,
			},
			"content": &hcldec.AttrSpec{
				Name: "content",
				Type: cty.String,
			},
			"file_mode": &hcldec.AttrSpec{
				Name: "file_mode",
				Type: cty.Number,
			},
			"dir_mode": &hcldec.AttrSpec{
				Name: "dir_mode",
				Type: cty.Number,
			},
			"ignore_dir_mode": &hcldec.AttrSpec{
				Name: "ignore_dir_mode",
				Type: cty.Bool,
			},
		},
		ctx,
	)
	if diags != nil {
		return diags
	}

	err := utils.FromCtyValue(value, t)
	if err != nil {
		return err
	}

	if value.GetAttr("condition").IsNull() {
		t.Condition = true
	}

	if t.Content != "" {
		t.Content = base64.StdEncoding.EncodeToString([]byte(t.Content))
	} else {
		src, err := utils.JoinCtyPath(body.(*hclsyntax.Body), t.Src)
		if err != nil {
			return err
		}

		data, err := iofs.ReadFile(src)
		if err != nil {
			return err
		}
		t.Content = base64.StdEncoding.EncodeToString(data)
	}

	t.Config = config
	t.value = value
	return nil
}

func (t *Task) Validate() error {
	if t.Src == "" && t.Content == "" {
		return errors.Errorf("Missing required argument; Either \"src\" or \"content\" is required.")
	}
	return nil
}

func (t *Task) Include() bool {
	return t.Condition
}

func (t *Task) Value() cty.Value {
	return t.value
}

func (t *Task) Apply() (any, error) {
	fileMode := t.FileMode
	if int(fileMode) == 0 {
		fileMode = t.Config.DefaultFileMode
	}

	dirMode := t.DirMode
	if int(dirMode) == 0 {
		dirMode = t.Config.DefaultDirMode
	}

	srcData, err := base64.StdEncoding.DecodeString(t.Content)
	if err != nil {
		return nil, err
	}

	dirChanges, err := Mkdir(filepath.Dir(t.Dst), dirMode, t.Config.DryRun)
	if err != nil {
		return nil, err
	}

	var permDirChanges []string
	if !t.IgnoreDirMode {
		permDirChanges, err = Chmod(filepath.Dir(t.Dst), dirMode, t.Config.DryRun)
		if err != nil {
			return nil, err
		}
	}

	fileChanges, err := WriteFile(t.Dst, srcData, fileMode, t.Config.DryRun)
	if err != nil {
		return nil, err
	}

	permFileChanges, err := Chmod(t.Dst, fileMode, t.Config.DryRun)
	if err != nil {
		return nil, err
	}

	return &Output{
		Changed: dirChanges != nil || fileChanges != nil || permDirChanges != nil || permFileChanges != nil,
		Diff: map[string][]string{
			"mkdir":       dirChanges,
			"file":        fileChanges,
			"permissions": append(permDirChanges, permFileChanges...),
		},
	}, nil
}

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
