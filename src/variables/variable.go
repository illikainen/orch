package variables

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
)

type Variable struct {
	Name         string   `hcl:"name,label"`
	Body         hcl.Body `hcl:"body,remain"`
	Dependencies []string
	value        cty.Value
}

func (v *Variable) PartialDecode() error {
	return nil
}

func (v *Variable) Decode(ctxfn func() (*hcl.EvalContext, error)) error {
	ctx, err := ctxfn()
	if err != nil {
		return err
	}

	value, diags := hcldec.Decode(
		v.Body,
		&hcldec.ObjectSpec{
			"value": &hcldec.AttrSpec{
				Name: "value",
				Type: cty.DynamicPseudoType,
			},
		},
		ctx,
	)
	if diags != nil {
		return diags
	}

	v.value = value
	return nil
}

func (v *Variable) Value() cty.Value {
	if v.value != cty.NilVal {
		return v.value.GetAttr("value")
	}
	return v.value
}

func (v *Variable) Unique() string {
	return v.Name
}

func (v *Variable) Validate() error {
	return nil
}
