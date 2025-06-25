package exec

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition bool            `json:"condition"`
	Cmd       string          `json:"cmd"`
	Shell     bool            `json:"shell"`
	Config    *configs.Config `json:"config"`
	value     cty.Value
}
