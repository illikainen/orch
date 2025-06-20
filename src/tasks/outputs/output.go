package outputs

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"

	"github.com/illikainen/orch/src/utils"
)

type Output struct {
	Type    string              `json:"type"`
	Host    string              `json:"host"`
	Role    string              `json:"role"`
	Name    string              `json:"name"`
	Changed bool                `json:"changed" cty:"changed"`
	Diff    map[string][]string `json:"diff"    cty:"diff"`
	Error   string              `json:"error"`
}

func (o *Output) IsChanged() bool {
	return o.Changed
}

func (o *Output) Differences() map[string][]string {
	return o.Diff
}

func (o *Output) Value() (cty.Value, error) {
	return utils.ToCtyValue(o)
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
	return nil
}
