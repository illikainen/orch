package qvm

import (
	"github.com/illikainen/orch/src/codec"
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/qubes"
	"github.com/illikainen/orch/src/tasks/decode"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/zclconf/go-cty/cty"
)

func init() {
	fn.Must(decode.Register("qvm", NewDecoder))
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

	values := value.AsValueMap()
	if prefs, ok := values["prefs"]; ok {
		values["prefs"] = qubes.HandleDefaultPreferences(prefs)
	}
	if services, ok := values["services"]; ok {
		values["services"] = qubes.HandleDefaultPreferences(services)
	}
	if features, ok := values["features"]; ok {
		values["features"] = qubes.HandleDefaultPreferences(features)
	}
	value = cty.ObjectVal(values)

	err = utils.FromCtyValue(value, d)
	if err != nil {
		return err
	}

	if value.GetAttr("condition").IsNull() {
		d.Condition = true
	}

	d.Preferences.Name = &d.Name
	d.Preferences.Label = &d.Label

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
