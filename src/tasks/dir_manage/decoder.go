//lint:ignore ST1003 readability
package dir_manage // revive:disable-line:var-naming

import (
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/hclang"
	"github.com/illikainen/orch/src/tasks/decode"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

func init() {
	fn.Must(decode.Register("dir_manage", NewDecoder))
}

type Decoder struct {
	Task
}

func NewDecoder() (decode.Decoder, error) {
	return &Decoder{
		Task: Task{
			Content: map[string]string{},
		},
	}, nil
}

func (d *Decoder) PartialDecode(body hcl.Body) error {
	_, err := hclang.Validate(body, d.Task, nil)
	if err != nil {
		return err
	}

	return nil
}

func (d *Decoder) Decode(body hcl.Body, ctx *hclang.EvalContext, config *configs.Config) error {
	value, err := hclang.Decode(body, &hclang.DecodeOptions{
		Context: ctx,
	})
	if err != nil {
		return err
	}

	err = hclang.FromCtyValue(value, d)
	if err != nil {
		return err
	}

	base, err := hclang.JoinCtyPath(body.(*hclsyntax.Body), d.Src)
	if err != nil {
		return err
	}

	err = filepath.WalkDir(base, func(src string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		exclude := seq.ContainsBy(d.Exclude, func(elt string) bool {
			pattern, e := hclang.JoinCtyPath(body.(*hclsyntax.Body), elt)
			if err != nil {
				err = e
				return false
			}
			match, e := filepath.Match(pattern, src)
			if e != nil {
				err = e
				return false
			}
			return match
		})
		if err != nil {
			return errors.WithStack(err)
		}
		if exclude {
			return nil
		}

		data, err := iofs.ReadFile(src)
		if err != nil {
			return err
		}

		dst := filepath.Join(d.Dst, src[len(base)+len(string(os.PathSeparator)):])
		d.Content[dst] = base64.StdEncoding.EncodeToString(data)
		return nil
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if int(d.FileMode) == 0 {
		d.FileMode = config.DefaultFileMode
	}

	if int(d.DirMode) == 0 {
		d.DirMode = config.DefaultDirMode
	}

	d.Config = config
	d.value = value
	return nil
}

func (d *Decoder) Dependencies(body hcl.Body) ([]string, error) {
	schema, err := hclang.GenerateBodySchema(d.Task, nil)
	if err != nil {
		return nil, err
	}

	return hclang.Dependencies(body, schema)
}

func (d *Decoder) Condition() bool {
	return d.Cond != nil && *d.Cond
}

func (d *Decoder) Value() cty.Value {
	return d.value
}
