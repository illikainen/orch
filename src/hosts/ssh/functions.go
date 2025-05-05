package ssh

import (
	"strings"

	"github.com/illikainen/go-netutils/src/sshx"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func (h *Host) Functions() map[string]function.Function {
	return map[string]function.Function{
		"exec": h.exec(),
	}
}
func (h *Host) exec() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "cmd",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			out, err := h.conn.Exec(&sshx.ExecOptions{
				Command: args[0].AsString(),
			})
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(strings.Trim(string(out.Stdout), "\r\n")), nil
		},
	})
}
