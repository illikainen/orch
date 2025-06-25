package exec

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition bool            `json:"condition" hcl:"condition,optional"`
	Cmd       string          `json:"cmd"       hcl:"cmd"`
	Shell     bool            `json:"shell"     hcl:"shell,optional"`
	Config    *configs.Config `json:"config"    hcl:"-"`
	value     cty.Value
}
