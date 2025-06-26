package systemd

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition bool            `json:"condition" hcl:"condition,optional"`
	Name      string          `json:"name"      hcl:"name"`
	Action    string          `json:"action"    hcl:"action"`
	Config    *configs.Config `json:"config"    hcl:"-"`
	value     cty.Value
}
