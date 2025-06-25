//lint:ignore ST1003 readability
package file_remove // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition bool            `json:"condition"`
	Path      string          `json:"path"`
	Config    *configs.Config `json:"config"`
	value     cty.Value
}
