package exec

import (
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/hclang"
	"github.com/illikainen/orch/src/tasks/decode"

	"github.com/hashicorp/hcl/v2"
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

func (d *Decoder) Include() bool {
	return d.Condition != nil && *d.Condition
}

func (d *Decoder) Value() cty.Value {
	return d.value
}
