package hclang

import (
	"reflect"
	"strings"

	"github.com/illikainen/go-utils/src/seq"
	"github.com/zclconf/go-cty/cty"
)

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
