package patch

import (
	"encoding/base64"
	"path/filepath"

	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/tasks/decode"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/zclconf/go-cty/cty"
)

func init() {
	fn.Must(decode.Register("patch", NewDecoder))
}

type Decoder struct {
	Task
}

func NewDecoder() (decode.Decoder, error) {
	return &Decoder{}, nil
}

func (d *Decoder) Decode(body hcl.Body, ctx *hcl.EvalContext, config *configs.Config) error {
	value, diags := hcldec.Decode(
		body,
		&hcldec.ObjectSpec{
			"condition": &hcldec.AttrSpec{
				Name: "condition",
				Type: cty.Bool,
			},
			"dir": &hcldec.AttrSpec{
				Name:     "dir",
				Type:     cty.String,
				Required: true,
			},
			"patch": &hcldec.AttrSpec{
				Name:     "patch",
				Type:     cty.String,
				Required: true,
			},
			"strip": &hcldec.AttrSpec{
				Name: "strip",
				Type: cty.Number,
			},
		},
		ctx,
	)
	if diags != nil {
		return diags
	}

	err := utils.FromCtyValue(value, d)
	if err != nil {
		return err
	}

	if value.GetAttr("condition").IsNull() {
		d.Condition = true
	}

	patch := d.Patch
	if !filepath.IsAbs(patch) {
		patch, err = utils.JoinCtyPath(body.(*hclsyntax.Body), patch)
		if err != nil {
			return err
		}
	}

	data, err := iofs.ReadFile(patch)
	if err != nil {
		return err
	}
	d.Content = base64.StdEncoding.EncodeToString(data)

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
