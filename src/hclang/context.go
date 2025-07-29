package hclang

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/illikainen/go-utils/src/assoc"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type EvalContextFunction interface {
	Functions() (map[string]function.Function, error)
}

type EvalContextVariable interface {
	Variables() (map[string]cty.Value, error)
}

type EvalContext struct {
	Variables []EvalContextVariable
	Functions []EvalContextFunction
}

func (e *EvalContext) Build() (*hcl.EvalContext, error) {
	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{},
		Functions: map[string]function.Function{},
	}

	for _, src := range e.Variables {
		vars, err := src.Variables()
		if err != nil {
			return nil, err
		}

		ctx.Variables, err = MergeCtyValues(ctx.Variables, vars)
		if err != nil {
			return nil, err
		}
	}

	for _, src := range e.Functions {
		fns, err := src.Functions()
		if err != nil {
			return nil, err
		}
		ctx.Functions = assoc.Merge(ctx.Functions, fns)
	}

	return ctx, nil
}
