package local

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/illikainen/orch/src/fact"
	"github.com/illikainen/orch/src/metadata"
	"github.com/illikainen/orch/src/tasks"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-utils/src/process"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type Host struct {
	Condition bool
	Hostname  string
	Become    string
	name      string
	bin       string
	value     cty.Value
}

func (h *Host) Decode(name string, body hcl.Body, ctx *hcl.EvalContext) error {
	value, diags := hcldec.Decode(
		body,
		&hcldec.ObjectSpec{
			"condition": &hcldec.AttrSpec{
				Name: "condition",
				Type: cty.Bool,
			},
			"hostname": &hcldec.AttrSpec{
				Name: "hostname",
				Type: cty.String,
			},
			"become": &hcldec.AttrSpec{
				Name: "become",
				Type: cty.String,
			},
		},
		ctx,
	)
	if diags != nil {
		return diags
	}

	err := utils.FromCtyValue(value, h)
	if err != nil {
		return err
	}

	if value.GetAttr("condition").IsNull() {
		h.Condition = true
	}

	h.name = name
	if h.Hostname == "" {
		h.Hostname = name
	}

	h.bin = filepath.Join(filepath.Join(".cache", metadata.Name()), "bin", metadata.Name())
	h.value = value

	return nil
}

func (h *Host) Validate() error {
	return nil
}

func (h *Host) Include() bool {
	return h.Condition
}

func (h *Host) Value() cty.Value {
	return h.value
}

func (h *Host) Dial() error {
	return nil
}

func (h *Host) Name() string {
	return h.name
}

func (h *Host) Close() error {
	return nil
}

func (h *Host) UploadBinary() error {
	return nil
}

func (h *Host) GatherFacts() (*fact.Facts, error) {
	facts, err := fact.GatherFacts()
	if err != nil {
		return nil, err
	}

	return facts, nil
}

func (h *Host) Functions() map[string]function.Function {
	return nil
}

func (h *Host) Apply(task *tasks.Task) (tasks.Outputter, error) {
	data, err := json.MarshalIndent(task, "", "    ")
	if err != nil {
		return nil, err
	}
	log.Tracef("local: apply %s", data)

	out, err := process.Exec(&process.ExecOptions{
		Command: []string{os.Args[0], "_apply-task"},
		Stdin:   bytes.NewReader(data),
		Become:  h.Become,
		Stderr:  process.LogrusOutput,
	})
	if err != nil {
		return nil, err
	}

	output := &tasks.Output{}
	err = json.Unmarshal(out.Stdout, output)
	if err != nil {
		return nil, err
	}

	return output, nil
}
