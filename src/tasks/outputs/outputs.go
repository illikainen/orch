package outputs

import (
	"encoding/json"

	"github.com/zclconf/go-cty/cty"
)

type Outputs []*Output

func (o *Outputs) Variables() (map[string]cty.Value, error) {
	outputs := map[string]cty.Value{}
	for _, out := range *o {
		if _, ok := outputs[out.Host]; !ok {
			outputs[out.Host] = cty.ObjectVal(map[string]cty.Value{})
		}
		hosts := outputs[out.Host].AsValueMap()
		if hosts == nil {
			hosts = map[string]cty.Value{}
		}

		if _, ok := hosts[out.Role]; !ok {
			hosts[out.Role] = cty.ObjectVal(map[string]cty.Value{})
		}
		roles := hosts[out.Role].AsValueMap()
		if roles == nil {
			roles = map[string]cty.Value{}
		}

		var err error
		roles[out.Name], err = out.Value()
		if err != nil {
			return nil, err
		}

		hosts[out.Role] = cty.ObjectVal(roles)
		outputs[out.Host] = cty.ObjectVal(hosts)
	}

	return map[string]cty.Value{"out": cty.ObjectVal(outputs)}, nil
}

func (o *Outputs) Clone() (Outputs, error) {
	// TODO: improve this...
	data, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	var out Outputs
	err = json.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (o *Outputs) Hosts() []string {
	hosts := []string{}
	for _, out := range *o {
		hosts = append(hosts, out.Host)
	}
	return hosts
}
