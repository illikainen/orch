package ssh

import (
	"strings"

	"github.com/illikainen/go-netutils/src/sshx"
	"github.com/kballard/go-shellquote"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func (h *Host) Functions() map[string]function.Function {
	return map[string]function.Function{
		"exists": h.exists(),
		"exec":   h.exec(),
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
			_, err := h.conn.Exec(&sshx.ExecOptions{
				Command: shellquote.Join("stat", "--", args[0].AsString()),
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
