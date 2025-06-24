package fact

import (
	"github.com/illikainen/orch/src/utils"

	"github.com/zclconf/go-cty/cty"
)

type Facts struct {
	Hostname string `cty:"hostname"`
	OS       *OS    `cty:"os"`
	IsQVM    bool   `cty:"is_qvm"`
}

func (f *Facts) Value() (cty.Value, error) {
	return utils.ToCtyValue(f)
}

func (f *Facts) Variables() (map[string]cty.Value, error) {
	value, err := f.Value()
	if err != nil {
		return nil, err
	}
	return map[string]cty.Value{"fact": value}, nil
}
