package patch

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition bool            `json:"condition" hcl:"condition,optional"`
	Dir       string          `json:"dir"       hcl:"dir"`
	Patch     string          `json:"patch"     hcl:"patch"`
	Strip     int             `json:"strip"     hcl:"strip"`
	Content   string          `json:"content"   hcl:"-"`
	Config    *configs.Config `json:"config"    hcl:"-"`
	value     cty.Value
}
