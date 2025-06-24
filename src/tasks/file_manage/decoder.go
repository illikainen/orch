//lint:ignore ST1003 readability
package file_manage // revive:disable-line:var-naming

import (
	"encoding/base64"

	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/tasks/decode"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

func init() {
	fn.Must(decode.Register("file_manage", NewDecoder))
}

type Decoder struct {
	Task
}

func NewDecoder() (decode.Decoder, error) {
	return &Decoder{}, nil
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
	if t.Src == "" && t.Content == "" {
		return errors.Errorf("Missing required argument; Either \"src\" or \"content\" is required.")
	}
	return nil
}

func (t *Decoder) Include() bool {
	return t.Condition
}

func (t *Decoder) Value() cty.Value {
	return t.value
}
