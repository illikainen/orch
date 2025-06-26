//lint:ignore ST1003 readability
package cert_unbundle // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/tasks/decode"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/zclconf/go-cty/cty"
)

func init() {
	fn.Must(decode.Register("cert_unbundle", NewDecoder))
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
				Name:     "src",
				Type:     cty.String,
				Required: true,
			},
			"dst": &hcldec.AttrSpec{
				Name:     "dst",
				Type:     cty.String,
				Required: true,
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
