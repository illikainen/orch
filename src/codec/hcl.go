package codec

import (
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/seq"
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

		name, optional, dynamic := parseFieldTag(field)
		if name == "-" {
			continue
		}

		kind := field.Type.Kind()
		if kind == reflect.Ptr {
			kind = field.Type.Elem().Kind()
		}

		if kind == reflect.Bool {
			spec[name] = &hcldec.AttrSpec{
				Name:     name,
				Type:     fn.Ternary(dynamic, cty.DynamicPseudoType, cty.Bool),
				Required: !optional,
			}
		} else if kind == reflect.Int64 {
			spec[name] = &hcldec.AttrSpec{
				Name:     name,
				Type:     fn.Ternary(dynamic, cty.DynamicPseudoType, cty.Number),
				Required: !optional,
			}
		} else if kind == reflect.String {
			spec[name] = &hcldec.AttrSpec{
				Name:     name,
				Type:     fn.Ternary(dynamic, cty.DynamicPseudoType, cty.String),
				Required: !optional,
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
			// NOTE: untested
			inner, err := GenerateObjectSpec(reflect.New(field.Type).Interface())
			if err != nil {
				return nil, err
			}

			spec[name] = &hcldec.BlockSpec{
				TypeName: name,
				Nested:   inner,
				Required: !optional,
			}
		} else {
			return nil, errors.Errorf("%s: unsupported type: %s", name, kind)
		}
	}

	return &spec, nil
}

func parseFieldTag(field reflect.StructField) (name string, optional bool, dynamic bool) {
	value := field.Tag.Get("orch")
	dynamic = seq.Contains(strings.Split(value, ","), "dynamic")

	value = field.Tag.Get("hcl")
	if value != "" {
		elts := strings.Split(value, ",")
		return elts[0], seq.Contains(elts[1:], "optional"), dynamic
	}

	value = field.Tag.Get("json")
	if value != "" {
		elts := strings.Split(value, ",")
		return elts[0], seq.Contains(elts[1:], "omitempty"), dynamic
	}

	return strings.ToLower(field.Name), false, dynamic
}
