package tasks

import (
	"encoding/json"

	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/rpc"
	"github.com/illikainen/orch/src/rpc/controller"
	"github.com/illikainen/orch/src/tasks/decode"
	_ "github.com/illikainen/orch/src/tasks/dir_manage"  // decoder
	_ "github.com/illikainen/orch/src/tasks/exec"        // decoder
	_ "github.com/illikainen/orch/src/tasks/file_manage" // decoder
	_ "github.com/illikainen/orch/src/tasks/file_remove" // decoder
	"github.com/illikainen/orch/src/tasks/outputs"
	_ "github.com/illikainen/orch/src/tasks/systemd_daemon_reload" // decoder

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type Task struct {
	Type         string          `json:"type"      hcl:"type,label"`
	Name         string          `json:"name"      hcl:"name,label"`
	Body         hcl.Body        `json:"-"         hcl:"body,remain"`
	Host         string          `json:"host"`
	Role         string          `json:"role"`
	Decoder      json.RawMessage `json:"decoder"`
	Dependencies []string        `json:"-"`
	decoder      decode.Decoder
}

func (t *Task) PartialDecode() error {
	attrs, diags := t.Body.JustAttributes()
	if diags != nil {
		return diags
	}

	for _, attr := range attrs {
		for _, v := range attr.Expr.Variables() {
			if len(v) >= 2 {
				if root, ok := v[0].(hcl.TraverseRoot); ok && root.Name == "out" {
					if host, ok := v[1].(hcl.TraverseAttr); ok && host.Name != "this" {
						t.Dependencies = append(t.Dependencies, host.Name)
					}
				}
			}
		}
	}

	return nil
}

func (t *Task) Decode(role string, host string, ctxfn func() (*hcl.EvalContext, error),
	config *configs.Config) error {
	ctx, err := ctxfn()
	if err != nil {
		return err
	}

	decoder, err := decode.Lookup(t.Type)
	if err != nil {
		return err
	}

	err = decoder.Decode(t.Body, ctx, config)
	if err != nil {
		return err
	}

	t.decoder = decoder
	t.Role = role
	t.Host = host

	return decoder.Validate()
}

func (t *Task) Validate() error {
	return nil
}

func (t *Task) Include() bool {
	return t.decoder.Include()
}

func (t *Task) Apply(ctrl *controller.Controller) (*outputs.Output, error) {
	rv, err := ctrl.Call(&rpc.FunctionCall{
		Function: t.Type,
		Params:   t.decoder,
	})
	if err != nil {
		return nil, err
	}

	var output outputs.Output
	err = json.Unmarshal(rv, &output)
	if err != nil {
		return nil, err
	}
	output.Type = t.Type
	output.Name = t.Name
	output.Host = t.Host
	output.Role = t.Role

	return &output, nil
}

func (t *Task) Unique() string {
	return t.Name
}

func (t *Task) Value() cty.Value {
	if t.decoder != nil {
		return t.decoder.Value()
	}
	return cty.NilVal
}

func (t *Task) MarshalJSON() ([]byte, error) {
	type alias Task
	task := alias(*t)

	decoder, err := json.Marshal(t.decoder)
	if err != nil {
		return nil, err
	}
	task.Decoder = decoder

	return json.Marshal(task)
}

func (t *Task) UnmarshalJSON(data []byte) error {
	type alias Task
	task := &alias{}

	err := json.Unmarshal(data, task)
	if err != nil {
		return err
	}
	*t = Task(*task)

	decoder, err := decode.Lookup(t.Type)
	if err != nil {
		return err
	}

	err = json.Unmarshal(t.Decoder, decoder)
	if err != nil {
		return err
	}
	t.decoder = decoder

	return nil
}
