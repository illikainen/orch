//lint:ignore ST1003 readability
package file_manage // revive:disable-line:var-naming

import (
	"os"

	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition     bool            `json:"condition"`
	Src           string          `json:"src"`
	Dst           string          `json:"dst"`
	Content       string          `json:"content"`
	FileMode      os.FileMode     `json:"file_mode"`
	DirMode       os.FileMode     `json:"dir_mode"`
	IgnoreDirMode bool            `json:"ignore_dir_mode"`
	Config        *configs.Config `json:"config"`
	value         cty.Value
}
