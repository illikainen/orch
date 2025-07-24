//lint:ignore ST1003 readability
package dir_manage // revive:disable-line:var-naming

import (
	"os"

	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition *bool             `json:"condition" hcl:"condition,optional" default:"true"`
	Src       string            `json:"src"       hcl:"src"`
	Dst       string            `json:"dst"       hcl:"dst"`
	Exclude   []string          `json:"exclude"   hcl:"exclude,optional"`
	FileMode  os.FileMode       `json:"file_mode" hcl:"file_mode,optional"`
	DirMode   os.FileMode       `json:"dir_mode"  hcl:"dir_mode,optional"`
	Content   map[string]string `json:"content"   hcl:"-"`
	Config    *configs.Config   `json:"config"    hcl:"-"`
	value     cty.Value
}
