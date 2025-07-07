//lint:ignore ST1003 readability
package qvm_firewall // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/qubes"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	qubes.Firewall
	Condition bool            `json:"condition" hcl:"condition,optional"`
	Name      string          `json:"name"      hcl:"name"`
	Config    *configs.Config `json:"config"    hcl:"-"`
	value     cty.Value
}
