//lint:ignore ST1003 readability
package file_remove // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition bool            `json:"condition" hcl:"condition,optional"`
	Path      string          `json:"path"      hcl:"path"`
	Config    *configs.Config `json:"config"    hcl:"-"`
	value     cty.Value
}
