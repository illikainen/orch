//lint:ignore ST1003 readability
package dir_manage // revive:disable-line:var-naming

import (
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"

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

func (t *Decoder) Decode(body hcl.Body, ctx *hcl.EvalContext, config *configs.Config) error {
	value, diags := hcldec.Decode(
		body,
		&hcldec.ObjectSpec{
			"condition": &hcldec.AttrSpec{
				Name: "condition",
				Type: cty.Bool,
			},
			"src": &hcldec.AttrSpec{
				Name:     "src",
				Type:     cty.String,
				Required: true,
			},
			"dst": &hcldec.AttrSpec{
				Name:     "dst",
				Type:     cty.String,
				Required: true,
			},
			"exclude": &hcldec.AttrSpec{
				Name: "exclude",
				Type: cty.List(cty.String),
			},
			"file_mode": &hcldec.AttrSpec{
				Name: "file_mode",
				Type: cty.Number,
			},
			"dir_mode": &hcldec.AttrSpec{
				Name: "dir_mode",
				Type: cty.Number,
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

	base, err := utils.JoinCtyPath(body.(*hclsyntax.Body), t.Src)
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

		exclude := seq.ContainsBy(t.Exclude, func(elt string) bool {
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

		dst := filepath.Join(t.Dst, src[len(base)+len(string(os.PathSeparator)):])
		t.Content[dst] = base64.StdEncoding.EncodeToString(data)
		return nil
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if int(t.FileMode) == 0 {
		t.FileMode = config.DefaultFileMode
	}

	if int(t.DirMode) == 0 {
		t.DirMode = config.DefaultDirMode
	}

	t.Config = config
	t.value = value
	return nil
}

func (t *Decoder) Validate() error {
	return nil
}

func (t *Decoder) Include() bool {
	return t.Condition
}

func (t *Decoder) Value() cty.Value {
	return t.value
}
