package patch

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition bool            `json:"condition"`
	Dir       string          `json:"dir"`
	Patch     string          `json:"patch"`
	Strip     int             `json:"strip"`
	Content   string          `json:"content"`
	Config    *configs.Config `json:"config"`
	value     cty.Value
}
