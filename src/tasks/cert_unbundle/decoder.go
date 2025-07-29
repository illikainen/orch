//lint:ignore ST1003 readability
package cert_unbundle // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/hclang"
	"github.com/illikainen/orch/src/tasks/decode"

	"github.com/hashicorp/hcl/v2"
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
