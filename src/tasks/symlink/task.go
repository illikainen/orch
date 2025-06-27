package symlink

import (
	"os"

	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition    bool            `json:"condition"`
	Src          string          `json:"src"`
	Dst          string          `json:"dst"`
	BaseDir      string          `json:"base_dir"`
	DirMode      os.FileMode     `json:"dir_mode"`
	LinkContents bool            `json:"link_contents"`
	Exclude      []string        `json:"exclude"`
	Config       *configs.Config `json:"config"`
	value        cty.Value
}
