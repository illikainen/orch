package hclang

import (
	"path"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/illikainen/go-utils/src/assoc"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
)

type BodySchema struct {
	Schema *hcl.BodySchema
	Blocks map[string]*BodySchema
}

func GenerateBodySchema(v any, opts *DecodeOptions) (*BodySchema, error) {
	if opts == nil {
		opts = &DecodeOptions{}
	}
	return generateBodySchema(v, opts, ".")
}

func generateBodySchema(v any, opts *DecodeOptions, location string) (*BodySchema, error) {
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
			if field.Type.Elem().Kind() == reflect.Struct || (field.Type.Elem().Kind() == reflect.Ptr &&
				field.Type.Elem().Elem().Kind() == reflect.Struct) {
				log.Tracef("%s: add '%s' as BlockHeaderSchema (slice)", typ.Name(), tags.Name)
				bs.Schema.Blocks = append(bs.Schema.Blocks, hcl.BlockHeaderSchema{
					Type: tags.Name,
				})

				inner, err := generateBodySchema(
					reflect.New(field.Type.Elem()).Interface(),
					opts,
					path.Join(location, tags.Name),
				)
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
			inner, err := generateBodySchema(
				reflect.New(field.Type).Interface(),
				opts,
				path.Join(location, tags.Name),
			)
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

	if seq.Contains(opts.ForEach, location) {
		bs.Schema.Attributes = append(bs.Schema.Attributes, hcl.AttributeSchema{
			Name:     "for_each",
			Required: false,
		})
	}
	return bs, nil
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
