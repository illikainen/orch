//lint:ignore ST1003 readability
package qvm_create // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/qubes"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition   bool              `json:"condition" hcl:"condition,optional"`
	Name        string            `json:"name"      hcl:"name"`
	Label       string            `json:"label"     hcl:"label"`
	Preferences qubes.Preferences `json:"prefs"     hcl:"prefs,optional"`
	Services    qubes.Services    `json:"services"  hcl:"services,optional"`
	Firewall    qubes.Firewall    `json:"firewall"  hcl:"firewall,optional"`
	Features    qubes.Features    `json:"features"  hcl:"features,optional"`
	Config      *configs.Config   `json:"config"    hcl:"-"`
	value       cty.Value
}
