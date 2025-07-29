//lint:ignore ST1003 readability
package file_manage // revive:disable-line:var-naming

import (
	"os"

	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Cond     *bool           `json:"condition" hcl:"condition,optional" default:"true"`
	Src      string          `json:"src"       hcl:"src,optional"       validate:"either:Content"`
	Dst      string          `json:"dst"       hcl:"dst"`
	Content  string          `json:"content"   hcl:"content,optional"`
	FileMode os.FileMode     `json:"file_mode" hcl:"file_mode,optional"`
	DirMode  os.FileMode     `json:"dir_mode"  hcl:"dir_mode,optional"`
	Config   *configs.Config `json:"config"    hcl:"-"`
	value    cty.Value
}
