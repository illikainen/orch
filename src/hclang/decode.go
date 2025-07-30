package hclang

import (
	"path"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/illikainen/go-utils/src/assoc"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type DecodeOptions struct {
	Body    hcl.Body
	Spec    *hcldec.ObjectSpec
	Context *EvalContext
	ForEach []string
}

func Decode(opts *DecodeOptions) (cty.Value, error) {
	if opts == nil {
		opts = &DecodeOptions{}
	}

	ctx, err := opts.Context.Build()
	if err != nil {
		return cty.NilVal, err
	}

	body, ok := opts.Body.(*hclsyntax.Body)
	if !ok {
		return cty.NilVal, errors.Errorf("bug")
	}

	values, err := decode(body, opts.Spec, ctx, ".", opts)
	if err != nil {
		return cty.NilVal, err
	}

	if !seq.Contains(opts.ForEach, ".") {
		if len(values) != 1 {
			return cty.NilVal, errors.Errorf("bug")
		}
		return values[0], nil
	}
	return cty.TupleVal(values), nil
}

func decode(body *hclsyntax.Body, spec *hcldec.ObjectSpec, ctx *hcl.EvalContext,
	location string, opts *DecodeOptions) ([]cty.Value, error) {
	var result []cty.Value
	if seq.Contains(opts.ForEach, location) && assoc.HasKey(body.Attributes, "for_each") {
		forEach, diags := body.Attributes["for_each"].Expr.Value(ctx)
		if diags != nil {
			return nil, diags
		}

		if !forEach.CanIterateElements() {
			return nil, errors.Errorf("for_each must be iterable")
		}

		it := forEach.ElementIterator()
		for it.Next() {
			_, each := it.Element()
			eachCtx := ctx.NewChild()
			if eachCtx.Variables == nil {
				eachCtx.Variables = map[string]cty.Value{}
			}
			eachCtx.Variables["each"] = each

			value, err := decodeBlock(body, spec, eachCtx, location, []string{"for_each"}, opts)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
	} else {
		value, err := decodeBlock(body, spec, ctx, location, nil, opts)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}

	return result, nil
}

func decodeBlock(body *hclsyntax.Body, spec *hcldec.ObjectSpec, ctx *hcl.EvalContext,
	location string, exclude []string, opts *DecodeOptions) (cty.Value, error) {
	values := map[string]cty.Value{}
	for name, attr := range body.Attributes {
		if !seq.Contains(exclude, name) {
			value, diags := attr.Expr.Value(ctx)
			if diags != nil {
				return cty.NilVal, errors.WithStack(diags)
			}
			values[name] = value
		}
	}

	blocks := map[string][]cty.Value{}
	for _, block := range body.Blocks {
		s, ok := (*spec)[block.Type]
		if !ok {
			return cty.NilVal, errors.Errorf("%s: unknown field", block.Type)
		}

		var blockSpec *hcldec.ObjectSpec
		switch s := s.(type) {
		case *hcldec.BlockSpec:
			blockSpec, ok = s.Nested.(*hcldec.ObjectSpec)
			if !ok {
				return cty.NilVal, errors.Errorf("%s: invalid type", block.Type)
			}
		case *hcldec.BlockListSpec:
			blockSpec, ok = s.Nested.(*hcldec.ObjectSpec)
			if !ok {
				return cty.NilVal, errors.Errorf("%s: invalid type", block.Type)
			}
		default:
			return cty.NilVal, errors.Errorf("%s: invalid type", block.Type)
		}

		blockValues, diags := decode(block.Body, blockSpec, ctx, path.Join(location, block.Type), opts)
		if diags != nil {
			return cty.NilVal, errors.WithStack(diags)
		}
		blocks[block.Type] = append(blocks[block.Type], blockValues...)
	}

	for name, block := range blocks {
		s, ok := (*spec)[name]
		if !ok {
			return cty.NilVal, errors.Errorf("%s: unknown field", name)
		}

		if _, ok := s.(*hcldec.BlockSpec); ok {
			if len(block) != 1 {
				return cty.NilVal, errors.Errorf("bug")
			}
			values[name] = block[0]
		} else {
			values[name] = cty.TupleVal(block)
		}
	}

	return cty.ObjectVal(values), nil
}
