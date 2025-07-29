package qvm

import (
	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/qubes"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Cond        *bool             `json:"condition" hcl:"condition,optional" default:"true"`
	Name        string            `json:"name"      hcl:"name"`
	Label       string            `json:"label"     hcl:"label"`
	Clone       *string           `json:"clone"     hcl:"clone,optional"`
	Preferences qubes.Preferences `json:"prefs"     hcl:"prefs,optional"`
	Services    qubes.Services    `json:"services"  hcl:"services,optional"`
	Firewall    qubes.Firewall    `json:"firewall"  hcl:"firewall,optional"`
	Features    qubes.Features    `json:"features"  hcl:"features,optional"`
	Tags        qubes.Tags        `json:"tags"      hcl:"tags,optional"`
	Volumes     []qubes.Volume    `json:"volume"    hcl:"volume,optional"`
	Config      *configs.Config   `json:"config"    hcl:"-"`
	value       cty.Value
}
