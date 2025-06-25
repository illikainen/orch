package exec

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
	fn.Must(decode.Register("exec", NewDecoder))
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
			"cmd": &hcldec.AttrSpec{
				Name:     "cmd",
				Type:     cty.String,
				Required: true,
			},
			"shell": &hcldec.AttrSpec{
				Name: "shell",
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
