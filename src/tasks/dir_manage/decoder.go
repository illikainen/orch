//lint:ignore ST1003 readability
package dir_manage // revive:disable-line:var-naming

import (
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/illikainen/orch/src/codec"
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/tasks/decode"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
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

func (d *Decoder) Decode(body hcl.Body, ctx *hcl.EvalContext, config *configs.Config) error {
	spec, err := codec.GenerateObjectSpec(d.Task)
	if err != nil {
		return err
	}

	value, diags := hcldec.Decode(body, spec, ctx)
	if diags != nil {
		return diags
	}

	err = utils.FromCtyValue(value, d)
	if err != nil {
		return err
	}

	if value.GetAttr("condition").IsNull() {
		d.Condition = true
	}

	base, err := utils.JoinCtyPath(body.(*hclsyntax.Body), d.Src)
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
			pattern, e := utils.JoinCtyPath(body.(*hclsyntax.Body), elt)
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

func (d *Decoder) Validate() error {
	return nil
}

func (d *Decoder) Include() bool {
	return d.Condition
}

func (d *Decoder) Value() cty.Value {
	return d.value
}
