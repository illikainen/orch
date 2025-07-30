package group

import (
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/hclang"
	"github.com/illikainen/orch/src/tasks/decode"

	"github.com/hashicorp/hcl/v2"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/zclconf/go-cty/cty"
)

func init() {
	fn.Must(decode.Register("group", NewDecoder))
}

type Decoder struct {
	Tasks  []Task          `json:"tasks"`
	Config *configs.Config `json:"config" hcl:"-"`
	value  cty.Value
}

func NewDecoder() (decode.Decoder, error) {
	return &Decoder{}, nil
}

func (d *Decoder) PartialDecode(body hcl.Body) error {
	var task Task
	_, err := hclang.Validate(body, task, nil)
	if err != nil {
		return err
	}

	return nil
}

func (d *Decoder) Decode(body hcl.Body, ctx *hclang.EvalContext, config *configs.Config) error {
	var task Task
	spec, err := hclang.GenerateObjectSpec(task)
	if err != nil {
		return err
	}

	value, err := hclang.Decode(&hclang.DecodeOptions{
		Body:    body,
		Spec:    spec,
		Context: ctx,
		ForEach: []string{"."},
	})
	if err != nil {
		return err
	}

	err = hclang.FromCtyValue(value, &d.Tasks)
	if err != nil {
		return err
	}

	d.Config = config
	d.value = value

	return nil
}

func (d *Decoder) Dependencies(body hcl.Body) ([]string, error) {
	var task Task
	schema, err := hclang.GenerateBodySchema(task, &hclang.DecodeOptions{
		Context: nil,
		ForEach: []string{"."},
	})
	if err != nil {
		return nil, err
	}

	return hclang.Dependencies(body, schema)
}

func (d *Decoder) Condition() bool {
	for _, task := range d.Tasks {
		if task.Cond != nil && *task.Cond {
			return true
		}
	}
	return false
}

func (d *Decoder) Value() cty.Value {
	return d.value
}
