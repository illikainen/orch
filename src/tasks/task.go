package tasks

import (
	"encoding/json"

	"github.com/illikainen/orch/src/configs"
	"github.com/illikainen/orch/src/tasks/file_manage"

	"github.com/hashicorp/hcl/v2"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type Runner interface {
	Decode(hcl.Body, *hcl.EvalContext, *configs.Config) error
	Validate() error
	Include() bool
	Value() cty.Value
	Apply() (any, error)
}

type Task struct {
	Type         string          `json:"type"      hcl:"type,label"`
	Name         string          `json:"name"      hcl:"name,label"`
	Body         hcl.Body        `json:"-"         hcl:"body,remain"`
	Host         string          `json:"host"`
	Role         string          `json:"role"`
	Runner       json.RawMessage `json:"runner"`
	Dependencies []string        `json:"-"`
	runner       Runner
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

	runner, err := t.getRunner()
	if err != nil {
		return err
	}

	err = runner.Decode(t.Body, ctx, config)
	if err != nil {
		return err
	}

	t.runner = runner
	t.Role = role
	t.Host = host

	return runner.Validate()
}

func (t *Task) Validate() error {
	return nil
}

func (t *Task) Include() bool {
	return t.runner.Include()
}

func (t *Task) Apply() (Outputter, error) {
	output, err := t.runner.Apply()
	if err != nil {
		return nil, err
	}

	outputter, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}

	return &Output{
		Type:      t.Type,
		Host:      t.Host,
		Role:      t.Role,
		Name:      t.Name,
		Outputter: outputter,
	}, nil
}

func (t *Task) Unique() string {
	return t.Name
}

func (t *Task) Value() cty.Value {
	if t.runner != nil {
		return t.runner.Value()
	}
	return cty.NilVal
}

func (t *Task) MarshalJSON() ([]byte, error) {
	type alias Task
	task := alias(*t)

	runner, err := json.Marshal(t.runner)
	if err != nil {
		return nil, err
	}
	task.Runner = runner

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

	runner, err := t.getRunner()
	if err != nil {
		return err
	}

	err = json.Unmarshal(t.Runner, runner)
	if err != nil {
		return err
	}
	t.runner = runner

	return nil
}

func (t *Task) getRunner() (Runner, error) {
	switch t.Type { // revive:disable-line:unnecessary-stmt
	case "file_manage":
		return &file_manage.Task{}, nil
	}
	return nil, errors.Errorf("%s is not a valid task type for %s", t.Type, t.Name)
}
