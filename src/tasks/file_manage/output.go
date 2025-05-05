//lint:ignore ST1003 readability
package file_manage // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/utils"

	"github.com/zclconf/go-cty/cty"
)

type Output struct {
	Changed bool                `json:"changed" cty:"changed"`
	Diff    map[string][]string `json:"diff"    cty:"diff"`
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
