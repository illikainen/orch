package bindings

import (
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/zclconf/go-cty/cty"
)

type Bindings []*Binding

func (b *Bindings) PartialDecode(basedir string) error {
	for _, binding := range *b {
		err := binding.PartialDecode(basedir)
		if err != nil {
			return err
		}
	}
	return b.Validate()
}

func (b *Bindings) Variables() map[string]cty.Value {
	roles := map[string]cty.Value{}
	bindings := map[string]cty.Value{}

	for _, binding := range *b {
		bindings[binding.Name] = binding.Value()
		for _, role := range binding.Roles {
			roles[role.Name] = role.Value()
		}
	}

	return map[string]cty.Value{
		"bind": cty.ObjectVal(bindings),
		"role": cty.ObjectVal(roles),
	}
}

func (b *Bindings) Validate() error {
	seen := []string{}
	for _, binding := range *b {
		if err := binding.Validate(); err != nil {
			return err
		}

		if lo.Contains(seen, binding.Unique()) {
			return errors.Errorf("\"%s\" is not unique", binding.Unique())
		}
		seen = append(seen, binding.Unique())
	}
	return nil
}
