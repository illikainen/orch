//lint:ignore ST1003 readability
package qvm_prefs // revive:disable-line:var-naming

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/illikainen/go-utils/src/fn"

	"github.com/illikainen/orch/src/qubes"
	"github.com/illikainen/orch/src/tasks/outputs"
)

func ApplyPreferences(name string, prefs *qubes.Preferences, dryRun bool) (int, []string, error) {
	if dryRun {
		exists, err := qubes.Check(name)
		if err != nil {
			return 0, nil, err
		}

		if !exists {
			return outputs.StatusIndeterminate, nil, nil
		}
	}

	updates, err := prefs.Apply(name, dryRun)
	if err != nil {
		return 0, nil, err
	}

	var changes []string
	for _, update := range updates {
		oldDefault := ""
		if update.OldIsDefault {
			oldDefault = " (default)"
		}

		newDefault := ""
		if update.NewIsDefault {
			newDefault = " (default)"
		}

		changes = append(changes, fmt.Sprintf("%s: %s%s -> %s%s", update.Property,
			update.OldValue, oldDefault, update.NewValue, newDefault))
	}

	return fn.Ternary(changes == nil, outputs.StatusUnchanged, outputs.StatusChanged), changes, nil
}

func HandleDefaultPreferences(v cty.Value) cty.Value {
	if !v.IsKnown() || v.IsNull() {
		return v
	}

	values := map[string]cty.Value{}
	var defaults []cty.Value

	for key, value := range v.AsValueMap() {
		if value.Type() == cty.String && !value.IsNull() && value.AsString() == "*default*" {
			defaults = append(defaults, cty.StringVal(key))
			values[key] = cty.NilVal
		} else {
			values[key] = value
		}
	}

	if defaults != nil {
		values["defaults"] = cty.ListVal(defaults)
	}
	return cty.ObjectVal(values)
}
