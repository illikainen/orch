package codec

import (
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type BodySchema struct {
	Schema *hcl.BodySchema
	Blocks map[string]*BodySchema
}

func GenerateBodySchema(v any) (*BodySchema, error) {
	spec, err := GenerateObjectSpec(v)
	if err != nil {
		return nil, err
	}

	body := &BodySchema{
		Schema: &hcl.BodySchema{},
		Blocks: map[string]*BodySchema{},
	}

	for name, spec := range *spec {
		if attr, ok := spec.(*hcldec.AttrSpec); ok {
			body.Schema.Attributes = append(body.Schema.Attributes, hcl.AttributeSchema{
				Name:     name,
				Required: attr.Required,
			})
		} else if _, ok := spec.(*hcldec.BlockSpec); ok {
			body.Schema.Blocks = append(body.Schema.Blocks, hcl.BlockHeaderSchema{
				Type: name,
			})

			field, ok := fieldByTagName(v, name)
			if !ok {
				return nil, errors.Errorf("%s: unknown field", name)
			}

			blockBody, err := GenerateBodySchema(reflect.New(field.Type).Interface())
			if err != nil {
				return nil, err
			}
			body.Blocks[name] = blockBody
		}
	}

	return body, nil
}

func fieldByTagName(v any, name string) (reflect.StructField, bool) {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		tags := parseFieldTags(field)
		if tags.Name == name {
			return field, true
		}

		if field.Type.Kind() == reflect.Struct && field.Anonymous {
			f, ok := fieldByTagName(reflect.New(field.Type).Interface(), name)
			if ok {
				return f, ok
			}
		}
	}

	return reflect.StructField{}, false
}

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

func Dependencies(body hcl.Body, schema *BodySchema) ([]string, error) {
	content, diags := body.Content(schema.Schema)
	if diags != nil {
		return nil, diags
	}

	var deps []string

	for _, attr := range content.Attributes {
		for _, v := range attr.Expr.Variables() {
			if len(v) >= 2 {
				if root, ok := v[0].(hcl.TraverseRoot); ok && root.Name == "out" {
					if host, ok := v[1].(hcl.TraverseAttr); ok && host.Name != "this" {
						deps = append(deps, host.Name)
					}
				}
			}
		}
	}

	for _, block := range content.Blocks {
		blockSchema, ok := schema.Blocks[block.Type]
		if !ok {
			return nil, errors.Errorf("%s: unknown block type", block.Type)
		}

		blockDeps, err := Dependencies(block.Body, blockSchema)
		if err != nil {
			return nil, err
		}

		deps = append(deps, blockDeps...)
	}

	return deps, nil
}
