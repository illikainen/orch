//lint:ignore ST1003 readability
package file_line // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/codec"
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition *bool           `json:"condition" hcl:"condition,optional" default:"true"`
	Path      string          `json:"path"      hcl:"path"`
	Line      string          `json:"line"      hcl:"line"`
	Regexp    *codec.Regexp   `json:"regexp"    hcl:"regexp"             orch:"string"`
	Config    *configs.Config `json:"config"    hcl:"-"`
	value     cty.Value
}
