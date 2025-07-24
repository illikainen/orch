package symlink

import (
	"os"

	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition    *bool           `json:"condition"     hcl:"condition,optional"     default:"true"`
	Src          string          `json:"src"           hcl:"src"`
	Dst          string          `json:"dst"           hcl:"dst"`
	DirMode      os.FileMode     `json:"dir_mode"      hcl:"dir_mode,optional"`
	LinkContents bool            `json:"link_contents" hcl:"link_contents,optional"`
	Exclude      []string        `json:"exclude"       hcl:"exclude,optional"`
	BaseDir      string          `json:"base_dir"      hcl:"-"`
	Config       *configs.Config `json:"config"        hcl:"-"`
	value        cty.Value
}
