package symlink

import (
	"github.com/illikainen/orch/src/codec"
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/tasks/decode"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/zclconf/go-cty/cty"
)

func init() {
	fn.Must(decode.Register("symlink", NewDecoder))
}

type Decoder struct {
	Task
}

func NewDecoder() (decode.Decoder, error) {
	return &Decoder{}, nil
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

	basedir, err := utils.CtyBaseDir(body.(*hclsyntax.Body))
	if err != nil {
		return err
	}
	d.BaseDir = basedir

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
	return d.Condition != nil && *d.Condition
}

func (d *Decoder) Value() cty.Value {
	return d.value
}
