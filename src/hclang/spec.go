package hclang

import (
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

func GenerateObjectSpec(v any) (*hcldec.ObjectSpec, error) {
	spec := hcldec.ObjectSpec{}

	typ := reflect.TypeOf(v)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		tags := parseFieldTags(field)
		if tags.Name == "-" {
			continue
		}

		kind := field.Type.Kind()
		if kind == reflect.Ptr {
			kind = field.Type.Elem().Kind()
		}

		if kind == reflect.Bool {
			spec[tags.Name] = &hcldec.AttrSpec{
				Name:     tags.Name,
				Type:     fn.Ternary(tags.Type != cty.NilType, tags.Type, cty.Bool),
				Required: tags.Required,
			}
		} else if kind == reflect.Uint32 || kind == reflect.Int || kind == reflect.Int64 {
			spec[tags.Name] = &hcldec.AttrSpec{
				Name:     tags.Name,
				Type:     fn.Ternary(tags.Type != cty.NilType, tags.Type, cty.Number),
				Required: tags.Required,
			}
		} else if kind == reflect.String {
			spec[tags.Name] = &hcldec.AttrSpec{
				Name:     tags.Name,
				Type:     fn.Ternary(tags.Type != cty.NilType, tags.Type, cty.String),
				Required: tags.Required,
			}
		} else if kind == reflect.Slice && field.Type.Elem().Kind() == reflect.String {
			spec[tags.Name] = &hcldec.AttrSpec{
				Name:     tags.Name,
				Type:     fn.Ternary(tags.Type != cty.NilType, tags.Type, cty.List(cty.String)),
				Required: tags.Required,
			}
		} else if kind == reflect.Struct && field.Anonymous {
			inner, err := GenerateObjectSpec(reflect.New(field.Type).Interface())
			if err != nil {
				return nil, err
			}

			for key, value := range *inner {
				spec[key] = value
			}
		} else if kind == reflect.Struct {
			// This is a workaround for wrappers like codec/regexp.go,
			// which should be a string attribute rather than a block.
			if tags.Type != cty.NilType {
				spec[tags.Name] = &hcldec.AttrSpec{
					Name:     tags.Name,
					Type:     tags.Type,
					Required: tags.Required,
				}
			} else {
				inner, err := GenerateObjectSpec(reflect.New(field.Type).Interface())
				if err != nil {
					return nil, err
				}

				spec[tags.Name] = &hcldec.BlockSpec{
					TypeName: tags.Name,
					Nested:   inner,
					Required: tags.Required,
				}
			}
		} else if kind == reflect.Slice && field.Type.Elem().Kind() == reflect.Struct {
			inner, err := GenerateObjectSpec(reflect.New(field.Type.Elem()).Interface())
			if err != nil {
				return nil, err
			}

			spec[tags.Name] = &hcldec.BlockListSpec{
				TypeName: tags.Name,
				Nested:   inner,
				MinItems: fn.Ternary(tags.Required, 1, 0),
			}
		} else if kind == reflect.Map {
			spec[tags.Name] = &hcldec.AttrSpec{
				Name:     tags.Name,
				Type:     fn.Ternary(tags.Type != cty.NilType, tags.Type, cty.DynamicPseudoType),
				Required: tags.Required,
			}
		} else {
			if kind == reflect.Interface {
				if field.Type.Implements(reflect.TypeOf((*hcl.Body)(nil)).Elem()) {
					continue
				}
			}
			return nil, errors.Errorf("%s: unsupported type: %s", tags.Name, kind)
		}
	}

	return &spec, nil
}
