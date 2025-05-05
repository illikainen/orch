package tasks

import (
	"encoding/json"

	"github.com/illikainen/orch/src/tasks/file_manage"

	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type Output struct {
	Type      string          `json:"type"`
	Host      string          `json:"host"`
	Role      string          `json:"role"`
	Name      string          `json:"name"`
	Outputter json.RawMessage `json:"runner"`
	Error     string          `json:"error"`
	outputter Outputter
}

type Outputter interface {
	IsChanged() bool
	Differences() map[string][]string
	Value() (cty.Value, error)
}

func (o *Output) IsChanged() bool {
	return o.outputter.IsChanged()
}

func (o *Output) Differences() map[string][]string {
	return o.outputter.Differences()
}

func (o *Output) Value() (cty.Value, error) {
	return o.outputter.Value()
}

func (o *Output) UnmarshalJSON(data []byte) error {
	type alias Output
	output := &alias{}

	err := json.Unmarshal(data, output)
	if err != nil {
		return err
	}

	if output.Error != "" {
		return errors.Errorf("%s", output.Error)
	}

	*o = Output(*output)

	outputter, err := o.getOutputter()
	if err != nil {
		return err
	}

	err = json.Unmarshal(o.Outputter, outputter)
	if err != nil {
		return err
	}

	o.outputter = outputter
	return nil
}

func (o *Output) getOutputter() (Outputter, error) {
	switch o.Type { // revive:disable-line:unnecessary-stmt
	case "file_manage":
		return &file_manage.Output{}, nil
	}
	return nil, errors.Errorf("%s is not a valid output type for %s", o.Type, o.Name)
}
