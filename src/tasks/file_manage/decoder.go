//lint:ignore ST1003 readability
package file_manage // revive:disable-line:var-naming

import (
	"encoding/base64"

	"github.com/illikainen/orch/src/codec"
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

	if d.Content != "" {
		d.Content = base64.StdEncoding.EncodeToString([]byte(d.Content))
	} else {
		src, err := utils.JoinCtyPath(body.(*hclsyntax.Body), d.Src)
		if err != nil {
			return err
		}

		data, err := iofs.ReadFile(src)
		if err != nil {
			return err
		}
		d.Content = base64.StdEncoding.EncodeToString(data)
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
	if d.Src == "" && d.Content == "" {
		return errors.Errorf("Missing required argument; Either \"src\" or \"content\" is required.")
	}
	return nil
}

func (d *Decoder) Include() bool {
	return d.Condition
}

func (d *Decoder) Value() cty.Value {
	return d.value
}
