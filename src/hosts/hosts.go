package hosts

import (
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/zclconf/go-cty/cty"
)

type Filter struct {
	Hosts []string
	Tags  []string
}

type Hosts []*Host

func (h *Hosts) PartialDecode(filter *Filter) error {
	hosts := Hosts{}

	for _, host := range *h {
		if (len(filter.Hosts) == 0 && len(filter.Tags) == 0) ||
			((len(filter.Hosts) == 0 || lo.Contains(filter.Hosts, host.Name)) &&
				(len(filter.Tags) == 0 || len(lo.Intersect(filter.Tags, host.Tags)) > 0)) {
			err := host.PartialDecode()
			if err != nil {
				return err
			}
			hosts = append(hosts, host)
		}
	}

	err := hosts.Validate()
	if err != nil {
		return err
	}

	*h = hosts
	return nil
}

func (h *Hosts) Variables() map[string]cty.Value {
	hosts := map[string]cty.Value{}
	for _, host := range *h {
		hosts[host.Name] = host.Value()
	}

	return map[string]cty.Value{"host": cty.ObjectVal(hosts)}
}

func (h *Hosts) Validate() error {
	seen := []string{}
	for _, host := range *h {
		if err := host.Validate(); err != nil {
			return err
		}

		if lo.Contains(seen, host.Unique()) {
			return errors.Errorf("\"%s\" is not unique", host.Unique())
		}
		seen = append(seen, host.Unique())
	}
	return nil
}
