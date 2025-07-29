package exec

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Cond   *bool           `json:"condition" hcl:"condition,optional" default:"true"`
	Cmd    string          `json:"cmd"       hcl:"cmd"`
	Shell  bool            `json:"shell"     hcl:"shell,optional"`
	Config *configs.Config `json:"config"    hcl:"-"`
	value  cty.Value
}
