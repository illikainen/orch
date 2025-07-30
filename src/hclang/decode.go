package hclang

import (
	"path"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/illikainen/go-utils/src/assoc"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type DecodeOptions struct {
	Context *EvalContext
	context *hcl.EvalContext
	ForEach []string
}

func Decode(body hcl.Body, opts *DecodeOptions) (cty.Value, error) {
	if opts == nil {
		opts = &DecodeOptions{}
	}

	if opts.Context != nil {
		ctx, err := opts.Context.Build()
		if err != nil {
			return cty.NilVal, err
		}
		opts.context = ctx
	}

	values, err := decode(body, opts, ".")
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

func decode(body hcl.Body, opts *DecodeOptions, location string) ([]cty.Value, error) {
	b, ok := body.(*hclsyntax.Body)
	if !ok {
		return nil, errors.Errorf("bug")
	}

	var result []cty.Value
	if seq.Contains(opts.ForEach, location) && assoc.HasKey(b.Attributes, "for_each") {
		forEach, diags := b.Attributes["for_each"].Expr.Value(opts.context)
		if diags != nil {
			return nil, diags
		}

		if !forEach.CanIterateElements() {
			return nil, errors.Errorf("for_each must be iterable")
		}

		it := forEach.ElementIterator()
		for it.Next() {
			_, each := it.Element()
			ctx := opts.context.NewChild()
			if ctx.Variables == nil {
				ctx.Variables = map[string]cty.Value{}
			}
			ctx.Variables["each"] = each

			values := map[string]cty.Value{}
			for name, attr := range b.Attributes {
				if name != "for_each" {
					value, diags := attr.Expr.Value(ctx)
					if diags != nil {
						return nil, diags
					}
					values[name] = value
				}
			}

			blocks := map[string][]cty.Value{}
			for _, block := range b.Blocks {
				o := *opts
				o.context = ctx
				blockValues, diags := decode(block.Body, &o, path.Join(location, block.Type))
				if diags != nil {
					return nil, diags
				}
				blocks[block.Type] = append(blocks[block.Type], blockValues...)
			}

			for name, block := range blocks {
				values[name] = cty.TupleVal(block)
			}

			result = append(result, cty.ObjectVal(values))
		}
	} else {
		values := map[string]cty.Value{}
		for name, attr := range b.Attributes {
			value, diags := attr.Expr.Value(opts.context)
			if diags != nil {
				return nil, diags
			}
			values[name] = value
		}

		blocks := map[string][]cty.Value{}
		for _, block := range b.Blocks {
			blockValues, diags := decode(block.Body, opts, path.Join(location, block.Type))
			if diags != nil {
				return nil, diags
			}
			blocks[block.Type] = append(blocks[block.Type], blockValues...)
		}

		for name, block := range blocks {
			values[name] = cty.TupleVal(block)
		}

		result = append(result, cty.ObjectVal(values))
	}

	return result, nil
}
