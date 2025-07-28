package hclang

import (
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-utils/src/assoc"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
)

type BodySchema struct {
	Schema *hcl.BodySchema
	Blocks map[string]*BodySchema
}

func GenerateBodySchema(v any) (*BodySchema, error) {
	bs := &BodySchema{
		Schema: &hcl.BodySchema{},
		Blocks: map[string]*BodySchema{},
	}

	typ := reflect.TypeOf(v)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	log.Tracef("%s: generating body schema...", typ.Name())

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

		switch kind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Bool, reflect.String:
			log.Tracef("%s: add '%s' as AttributeSchema", typ.Name(), tags.Name)
			bs.Schema.Attributes = append(bs.Schema.Attributes, hcl.AttributeSchema{
				Name:     tags.Name,
				Required: tags.Required,
			})
			continue
		case reflect.Slice:
			if field.Type.Elem().Kind() == reflect.Struct {
				log.Tracef("%s: add '%s' as BlockHeaderSchema (slice)", typ.Name(), tags.Name)
				bs.Schema.Blocks = append(bs.Schema.Blocks, hcl.BlockHeaderSchema{
					Type: tags.Name,
				})

				inner, err := GenerateBodySchema(reflect.New(field.Type.Elem()).Interface())
				if err != nil {
					return nil, err
				}
				bs.Blocks[tags.Name] = inner
			} else {
				log.Tracef("%s: add '%s' as AttributeSchema (slice)", typ.Name(), tags.Name)
				bs.Schema.Attributes = append(bs.Schema.Attributes, hcl.AttributeSchema{
					Name:     tags.Name,
					Required: tags.Required,
				})
			}
			continue
		case reflect.Struct:
			inner, err := GenerateBodySchema(reflect.New(field.Type).Interface())
			if err != nil {
				return nil, err
			}

			if field.Anonymous {
				log.Tracef("%s: add '%s' as an embedded struct", typ.Name(), tags.Name)
				bs.Schema.Attributes = append(bs.Schema.Attributes, inner.Schema.Attributes...)
				bs.Schema.Blocks = append(bs.Schema.Blocks, inner.Schema.Blocks...)
				bs.Blocks = assoc.Merge(bs.Blocks, inner.Blocks)
			} else if tags.Type != cty.NilType {
				log.Tracef("%s: add '%s' as AttributeSchema (override)", typ.Name(), tags.Name)
				bs.Schema.Attributes = append(bs.Schema.Attributes, hcl.AttributeSchema{
					Name:     tags.Name,
					Required: tags.Required,
				})
			} else {
				log.Tracef("%s: add '%s' as BlockHeaderSchema", typ.Name(), tags.Name)
				bs.Schema.Blocks = append(bs.Schema.Blocks, hcl.BlockHeaderSchema{
					Type: tags.Name,
				})
				bs.Blocks[tags.Name] = inner
			}
			continue
		}

		return nil, errors.Errorf("%s: %s: unsupported body field type: %s", typ.Name(), field.Name, kind)
	}

	return bs, nil
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

func Validate(body hcl.Body, v any, opts *DecodeOptions) (*hcl.BodyContent, error) {
	bs, err := GenerateBodySchema(v, opts)
	if err != nil {
		return nil, err
	}

	content, diags := body.Content(bs.Schema)
	if diags != nil {
		return nil, err
	}

	return content, nil
}
