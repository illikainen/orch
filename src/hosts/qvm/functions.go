package qvm

import (
	"github.com/illikainen/orch/src/qubes"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func (h *Host) Functions() map[string]function.Function {
	return map[string]function.Function{
		"exists": h.exists(),
	}
}

func (h *Host) exists() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "path",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.Bool),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			_, err := qubes.Exec(&qubes.ExecOptions{
				Name:    h.Hostname,
				Command: []string{"stat", "--", args[0].AsString()},
				Become:  h.Become,
			})
			if err != nil {
				// FIXME: check the exit code and return non-ENOENT
				//lint:ignore nilerr skipped
				return cty.BoolVal(false), nil
			}

			return cty.BoolVal(true), nil
		},
	})
}
