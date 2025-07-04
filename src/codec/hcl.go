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
		} else {
			return nil, errors.Errorf("%s: unsupported type: %s", tags.Name, kind)
		}
	}

	return &spec, nil
}

type tags struct {
	Name     string
	Required bool
	Type     cty.Type
}

func parseFieldTags(field reflect.StructField) *tags {
	orch := strings.Split(field.Tag.Get("orch"), ",")
	hc := strings.Split(field.Tag.Get("hcl"), ",")
	js := strings.Split(field.Tag.Get("json"), ",")

	t := &tags{}

	if hc[0] != "" {
		t.Name = hc[0]
	} else if js[0] != "" {
		t.Name = js[0]
	} else {
		t.Name = strings.ToLower(field.Name)
	}

	if !seq.Contains(hc[1:], "optional") && !seq.Contains(js[1:], "omitempty") {
		t.Required = true
	}

	if seq.Contains(orch, "dynamic") {
		t.Type = cty.DynamicPseudoType
	} else if seq.Contains(orch, "string") {
		t.Type = cty.String
	}

	return t
}
