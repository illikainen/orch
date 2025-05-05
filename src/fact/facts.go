package fact

import (
	"os"

	"github.com/illikainen/orch/src/utils"

	"github.com/zclconf/go-cty/cty"
)

type Facts struct {
	Hostname string `cty:"hostname"`
	OS       *OS    `cty:"os"`
}

func GatherFacts() (*Facts, error) {
	facts := &Facts{}

	var err error
	facts.Hostname, err = os.Hostname()
	if err != nil {
		return nil, err
	}

	facts.OS, err = GatherOSFacts()
	if err != nil {
		return nil, err
	}

	return facts, nil
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
