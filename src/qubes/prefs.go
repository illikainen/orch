package qubes

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/process"
	"github.com/zclconf/go-cty/cty"

	"github.com/illikainen/orch/src/codec"
	"github.com/illikainen/orch/src/tasks/outputs"
)

type Preferences struct { // revive:disable:line-length-limit
	AudioVM            *string          `json:"audiovm,omitempty"              hcl:"audiovm,optional"              orch:"dynamic"`
	AutoCleanup        *bool            `json:"auto_cleanup,omitempty"         hcl:"auto_cleanup,optional"         orch:"dynamic"`
	AutoStart          *bool            `json:"autostart,omitempty"            hcl:"autostart,optional"            orch:"dynamic"`
	BackupTimestamp    *codec.EpochTime `json:"backup_timestamp,omitempty"     hcl:"backup_timestamp,optional"     orch:"dynamic"`
	Class              *string          `json:"class,omitempty"                hcl:"class,optional"                orch:"dynamic"`
	DNS                *string          `json:"dns,omitempty"                  hcl:"dns,optional"                  orch:"dynamic"`
	Debug              *bool            `json:"debug,omitempty"                hcl:"debug,optional"                orch:"dynamic"`
	DefaultDispVM      *string          `json:"default_dispvm,omitempty"       hcl:"default_dispvm,optional"       orch:"dynamic"`
	DefaultUser        *string          `json:"default_user,omitempty"         hcl:"default_user,optional"         orch:"dynamic"`
	GUIVM              *string          `json:"guivm,omitempty"                hcl:"guivm,optional"                orch:"dynamic"`
	Gateway            *string          `json:"gateway,omitempty"              hcl:"gateway,optional"              orch:"dynamic"`
	Gateway6           *string          `json:"gateway6,omitempty"             hcl:"gateway6,optional"             orch:"dynamic"`
	IP                 *string          `json:"ip,omitempty"                   hcl:"ip,optional"                   orch:"dynamic"`
	IP6                *string          `json:"ip6,omitempty"                  hcl:"ip6,optional"                  orch:"dynamic"`
	Icon               *string          `json:"icon,omitempty"                 hcl:"icon,optional"                 orch:"dynamic"`
	IncludeInBackups   *bool            `json:"include_in_backups,omitempty"   hcl:"include_in_backups,optional"   orch:"dynamic"`
	Kernel             *string          `json:"kernel,omitempty"               hcl:"kernel,optional"               orch:"dynamic"`
	KernelOpts         *string          `json:"kernelopts,omitempty"           hcl:"kernelopts,optional"           orch:"dynamic"`
	KeyboardLayout     *string          `json:"keyboard_layout,omitempty"      hcl:"keyboard_layout,optional"      orch:"dynamic"`
	Label              *string          `json:"label,omitempty"                hcl:"label,optional"`
	MAC                *string          `json:"mac,omitempty"                  hcl:"mac,optional"                  orch:"dynamic"`
	ManagementDispVM   *string          `json:"management_dispvm,omitempty"    hcl:"management_dispvm,optional"    orch:"dynamic"`
	MaxMem             *int64           `json:"maxmem,omitempty"               hcl:"maxmem,optional"               orch:"dynamic"`
	Memory             *int64           `json:"memory,omitempty"               hcl:"memory,optional"               orch:"dynamic"`
	Name               *string          `json:"name,omitempty"                 hcl:"name,optional"`
	NetVM              *string          `json:"netvm,omitempty"                hcl:"netvm,optional"                orch:"dynamic"`
	ProvidesNetwork    *bool            `json:"provides_network,omitempty"     hcl:"provides_network,optional"     orch:"dynamic"`
	QrexecTimeout      *int64           `json:"qrexec_timeout,omitempty"       hcl:"qrexec_timeout,optional"       orch:"dynamic"`
	ShutdownTimeout    *int64           `json:"shutdown_timeout,omitempty"     hcl:"shutdown_timeout,optional"     orch:"dynamic"`
	StartTime          *codec.EpochTime `json:"start_time,omitempty"           hcl:"start_time,optional"           orch:"dynamic"`
	Template           *string          `json:"template,omitempty"             hcl:"template,optional"             orch:"dynamic"`
	TemplateForDispVMs *bool            `json:"template_for_dispvms,omitempty" hcl:"template_for_dispvms,optional" orch:"dynamic"`
	UUID               *string          `json:"uuid,omitempty"                 hcl:"uuid,optional"                 orch:"dynamic"`
	Updateable         *bool            `json:"updateable,omitempty"           hcl:"updateable,optional"           orch:"dynamic"`
	VCPUs              *int64           `json:"vcpus,omitempty"                hcl:"vcpus,optional"                orch:"dynamic"`
	VirtMode           *string          `json:"virt_mode,omitempty"            hcl:"virt_mode,optional"            orch:"dynamic"`
	VisibleGateway     *string          `json:"visible_gateway,omitempty"      hcl:"visible_gateway,optional"      orch:"dynamic"`
	VisibleGateway6    *string          `json:"visible_gateway6,omitempty"     hcl:"visible_gateway6,optional"     orch:"dynamic"`
	VisibleIP          *string          `json:"visible_ip,omitempty"           hcl:"visible_ip,optional"           orch:"dynamic"`
	VisibleIP6         *string          `json:"visible_ip6,omitempty"          hcl:"visible_ip6,optional"          orch:"dynamic"`
	VisibleNetmask     *string          `json:"visible_netmask,omitempty"      hcl:"visible_netmask,optional"      orch:"dynamic"`
	Defaults           []string         `json:"defaults,omitempty"             hcl:"-"`
} // revive:enable:line-length-limit

type PreferenceChange struct {
	Property     string `json:"property"`
	OldValue     string `json:"old_value"`
	OldIsDefault bool   `json:"old_is_default"`
	NewValue     string `json:"new_value"`
	NewIsDefault bool   `json:"new_is_default"`
}

//go:embed prefs.py
var pythonPrefs []byte

func (p *Preferences) Apply(name string, dryRun bool) (status int, changes []string, err error) {
	if dryRun {
		exists, err := Check(name)
		if err != nil {
			return 0, nil, err
		}

		if !exists {
			return outputs.StatusIndeterminate, nil, nil
		}
	}

	data, err := json.Marshal(p)
	if err != nil {
		return 0, nil, err
	}

	tmp, tmpClean, err := iofs.MkdirTemp()
	if err != nil {
		return 0, nil, err
	}
	defer errorx.Defer(tmpClean, &err)

	prefs := filepath.Join(tmp, "prefs.py")
	err = iofs.WriteFile(prefs, bytes.NewReader(pythonPrefs))
	if err != nil {
		return 0, nil, err
	}

	cmd := []string{"python3", prefs, "set"}
	if dryRun {
		cmd = append(cmd, "--dry-run")
	}
	cmd = append(cmd, "--", name)

	proc, err := process.Exec(&process.ExecOptions{
		Command: cmd,
		Stdin:   bytes.NewReader(data),
	})
	if err != nil {
		return 0, nil, err
	}

	var prefChanges []PreferenceChange
	err = json.Unmarshal(proc.Stdout, &changes)
	if err != nil {
		return 0, nil, err
	}

	for _, change := range prefChanges {
		oldDefault := ""
		if change.OldIsDefault {
			oldDefault = " (default)"
		}

		newDefault := ""
		if change.NewIsDefault {
			newDefault = " (default)"
		}

		changes = append(changes, fmt.Sprintf("%s: %s%s -> %s%s", change.Property,
			change.OldValue, oldDefault, change.NewValue, newDefault))
	}

	return fn.Ternary(changes == nil, outputs.StatusUnchanged, outputs.StatusChanged), changes, err
}

func HandleDefaultPreferences(v cty.Value) cty.Value {
	if !v.IsKnown() || v.IsNull() {
		return v
	}

	values := map[string]cty.Value{}
	var defaults []cty.Value

	for key, value := range v.AsValueMap() {
		if value.Type() == cty.String && !value.IsNull() && value.AsString() == "*default*" {
			defaults = append(defaults, cty.StringVal(key))
			values[key] = cty.NilVal
		} else {
			values[key] = value
		}
	}

	if defaults != nil {
		values["defaults"] = cty.ListVal(defaults)
	}
	return cty.ObjectVal(values)
}
