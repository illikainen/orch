package variables

import (
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"

	"github.com/illikainen/orch/src/hclang"
)

type Variables []*Variable

func (v *Variables) PartialDecode() error {
	for _, elt := range *v {
		if err := elt.PartialDecode(); err != nil {
			return err
		}
	}
	return nil
}

func (v *Variables) Decode(ctx *hclang.EvalContext) error {
	for _, elt := range *v {
		if err := elt.Decode(ctx); err != nil {
			return err
		}
	}
	return v.Validate()
}

func (v *Variables) Dependencies() []string {
	deps := []string{}
	for _, elt := range *v {
		deps = append(deps, elt.Dependencies...)
	}
	return deps
}

func (v *Variables) Variables() (map[string]cty.Value, error) {
	vars := map[string]cty.Value{}
	for _, variable := range *v {
		vars[variable.Name] = variable.Value()
	}

	return map[string]cty.Value{"var": cty.ObjectVal(vars)}, nil
}

func (v *Variables) Validate() error {
	seen := []string{}
	for _, elt := range *v {
		if err := elt.Validate(); err != nil {
			return err
		}

		if seq.Contains(seen, elt.Unique()) {
			return errors.Errorf("\"%s\" is not unique", elt.Unique())
		}
		seen = append(seen, elt.Unique())
	}
	return nil
}
