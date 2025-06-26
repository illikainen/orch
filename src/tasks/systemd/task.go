package systemd

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition bool            `json:"condition"`
	Name      string          `json:"name"`
	Action    string          `json:"action"`
	Config    *configs.Config `json:"config"`
	value     cty.Value
}
