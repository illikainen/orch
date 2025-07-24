//lint:ignore ST1003 readability
package systemd_daemon_reload // revive:disable-line:var-naming

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Condition *bool           `json:"condition" hcl:"condition,optional" default:"true"`
	Config    *configs.Config `json:"config"    hcl:"-"`
	value     cty.Value
}
