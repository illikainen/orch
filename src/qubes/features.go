package qubes

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"

	"github.com/illikainen/orch/src/codec"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/google/go-cmp/cmp"
	"github.com/illikainen/go-utils/src/process"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type Features struct { // revive:disable:line-length-limit
	UpdatesAvailable *bool    `json:"updates_available,omitempty" hcl:"updates_available,optional" mapstructure:"updates-available" orch:"dynamic"`
	WhonixGW         *int64   `json:"whonix_gw,omitempty"         hcl:"whonix_gw,optional"         mapstructure:"whonix-gw"         orch:"dynamic"`
	WhonixWS         *int64   `json:"whonix_ws,omitempty"         hcl:"whonix_ws,optional"         mapstructure:"whonix-ws"         orch:"dynamic"`
	Defaults         []string `json:"defaults,omitempty"          hcl:"-"`
} // revive:enable:line-length-limit

func GetFeatures(name string) (*Features, error) {
	rx, err := regexp.Compile(`^([a-zA-Z0-9.-]+)[ \t]+([a-zA-Z0-9@:=?+/.,_ -]+)$`)
	if err != nil {
		return nil, err
	}

	p, err := process.Exec(&process.ExecOptions{
		Command: []string{"qvm-features", "--", name},
	})
	if err != nil {
		return nil, err
	}

	raw := map[string]string{}
	scan := bufio.NewScanner(bytes.NewReader(p.Stdout))
	for scan.Scan() {
		m := rx.FindStringSubmatch(scan.Text())
		if len(m) != 3 {
			return nil, errors.Errorf("invalid qvm-features line: '%s'", scan.Text())
		}

		raw[m[1]] = m[2]
	}

	err = scan.Err()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	features := Features{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:           codec.StringToPrimitiveHookFunc(),
		Result:               &features,
		IgnoreUntaggedFields: true,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = decoder.Decode(raw)
	if err != nil {
		return nil, err
	}

	return &features, err
}

func (f *Features) Apply(name string, dryRun bool) (int, []string, error) {
	cur, err := GetFeatures(name)
	if err != nil {
		return 0, nil, err
	}

	diff := &Diff{defaults: f.Defaults}
	cmp.Equal(cur, f, cmp.Reporter(diff))

	var changes []string
	for _, change := range diff.changes {
		orig := "*default*"
		if change.Old != nil {
			orig = *change.Old
		}

		next := "*default*"
		if change.New != nil {
			next = *change.New
			if !dryRun {
				_, err := process.Exec(&process.ExecOptions{
					Command: []string{"qvm-features", "--", name, change.Name, *change.New},
				})
				if err != nil {
					return 0, nil, err
				}
			}
		} else {
			if !dryRun {
				_, err := process.Exec(&process.ExecOptions{
					Command: []string{"qvm-features", "--default", "--", name, change.Name},
				})
				if err != nil {
					return 0, nil, err
				}
			}
		}

		changes = append(changes, fmt.Sprintf("%s: %s -> %s", change.Name, orig, next))
	}

	status := outputs.StatusUnchanged
	if changes != nil {
		status = outputs.StatusChanged
	}

	return status, changes, nil
}
