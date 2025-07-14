package qubes

import (
	"bufio"
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/process"
	"github.com/pkg/errors"
)

type Services struct { // revive:disable:line-length-limit
	MeminfoWriter       *bool    `json:"meminfo_writer,omitempty"        hcl:"meminfo_writer,optional"        orch:"dynamic"`
	QubesFirewall       *bool    `json:"qubes_firewall,omitempty"        hcl:"qubes_firewall,optional"        orch:"dynamic"`
	QubesNetwork        *bool    `json:"qubes_network,omitempty"         hcl:"qubes_network,optional"         orch:"dynamic"`
	QubesUpdateCheck    *bool    `json:"qubes_update_check,omitempty"    hcl:"qubes_update_check,optional"    orch:"dynamic"`
	Cups                *bool    `json:"cups,omitempty"                  hcl:"cups,optional"                  orch:"dynamic"`
	Crond               *bool    `json:"crond,omitempty"                 hcl:"crond,optional"                 orch:"dynamic"`
	NetworkManager      *bool    `json:"network_manager,omitempty"       hcl:"network_manager,optional"       orch:"dynamic"`
	Clocksync           *bool    `json:"clocksync,omitempty"             hcl:"clocksync,optional"             orch:"dynamic"`
	QubesUpdatesProxy   *bool    `json:"qubes_updates_proxy,omitempty"   hcl:"qubes_updates_proxy,optional"   orch:"dynamic"`
	UpdatesProxySetup   *bool    `json:"updates_proxy_setup,omitempty"   hcl:"updates_proxy_setup,optional"   orch:"dynamic"`
	DisableDefaultRoute *bool    `json:"disable_default_route,omitempty" hcl:"disable_default_route,optional" orch:"dynamic"`
	DisableDNSServer    *bool    `json:"disable_dns_server,omitempty"    hcl:"disable_dns_server,optional"    orch:"dynamic"`
	LightDM             *bool    `json:"lightdm,omitempty"               hcl:"lightdm,optional"               orch:"dynamic"`
	SoftwareRendering   *bool    `json:"software_rendering,omitempty"    hcl:"software_rendering,optional"    orch:"dynamic"`
	GuiVM               *bool    `json:"guivm,omitempty"                 hcl:"guivm,optional"                 orch:"dynamic"`
	GuiVMGuiAgent       *bool    `json:"guivm_gui_agent,omitempty"       hcl:"guivm_gui_agent,optional"       orch:"dynamic"`
	GuiVMVNC            *bool    `json:"guivm_vnc,omitempty"             hcl:"guivm_vnc,optional"             orch:"dynamic"`
	Tracker             *bool    `json:"tracker,omitempty"               hcl:"tracker,optional"               orch:"dynamic"`
	EvolutionDataServer *bool    `json:"evolution_data_server,omitempty" hcl:"evolution_data_server,optional" orch:"dynamic"`
	USBResetOnAttach    *bool    `json:"usb_reset_on_attach,omitempty"   hcl:"usb_reset_on_attach,optional"   orch:"dynamic"`
	Defaults            []string `json:"defaults,omitempty"              hcl:"-"`
} // revive:enable:line-length-limit

func (s *Services) Apply(name string, dryRun bool) (int, []string, error) {
	if dryRun {
		exists, err := Check(name)
		if err != nil {
			return 0, nil, err
		}
		if !exists {
			return outputs.StatusIndeterminate, nil, nil
		}
	}

	svc, err := services(name)
	if err != nil {
		return 0, nil, err
	}

	arg := map[bool]string{
		true:  "--enable",
		false: "--disable",
	}
	var changes []string

	t := reflect.TypeOf(*s)
	v := reflect.ValueOf(*s)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Bool && !value.IsNil() {
			fieldName := strings.ReplaceAll(strings.Split(field.Tag.Get("json"), ",")[0], "_", "-")
			if fieldName == "" {
				return 0, nil, errors.Errorf("invalid json tag name for: %s", field.Name)
			}
			fieldValue := value.Elem().Bool()

			curValue, ok := svc[fieldName]
			if !ok || curValue != fieldValue {
				if !dryRun {
					_, err := process.Exec(&process.ExecOptions{
						Command: []string{"qvm-service", arg[fieldValue], "--", name, fieldName},
					})
					if err != nil {
						return 0, nil, err
					}
				}

				if ok {
					changes = append(changes, fmt.Sprintf("%s: %t -> %t", fieldName, curValue, fieldValue))
				} else {
					changes = append(changes, fmt.Sprintf("%s: *default* -> %t", fieldName, fieldValue))
				}
			}
		}
	}

	for _, def := range s.Defaults {
		fieldName := strings.ReplaceAll(def, "_", "-")
		curValue, ok := svc[fieldName]
		if ok {
			if !dryRun {
				_, err := process.Exec(&process.ExecOptions{
					Command: []string{"qvm-service", "--default", "--", name, fieldName},
				})
				if err != nil {
					return 0, nil, err
				}
			}
			changes = append(changes, fmt.Sprintf("%t -> *default*", curValue))
		}
	}

	return fn.Ternary(changes == nil, outputs.StatusUnchanged, outputs.StatusChanged), changes, nil
}

func services(name string) (map[string]bool, error) {
	p, err := process.Exec(&process.ExecOptions{
		Command: []string{"qvm-service", "--", name},
	})
	if err != nil {
		return nil, err
	}

	rx, err := regexp.Compile(`^([a-z-]+)[ \t]+(on|off)$`)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rv := map[string]bool{}
	scan := bufio.NewScanner(bytes.NewReader(p.Stdout))
	for scan.Scan() {
		m := rx.FindStringSubmatch(scan.Text())
		if len(m) != 3 {
			return nil, errors.Errorf("invalid qvm-service line: '%s'", scan.Text())
		}

		rv[m[1]] = m[2] == "on"
	}

	err = scan.Err()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return rv, nil
}
