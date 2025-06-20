package local

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/illikainen/orch/src/metadata"
	"github.com/illikainen/orch/src/rpc/controller"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
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
	cmd       *exec.Cmd
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

func (h *Host) UploadBinary() error {
	return nil
}

func (h *Host) Start() (*controller.Controller, error) {
	bin, err := os.Executable()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(bin, "_rpc") // #nosec G204
	w, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	r, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	ctrl := controller.New(r, w)
	err = ctrl.Start()
	if err != nil {
		return nil, err
	}

	h.cmd = cmd
	return ctrl, nil
}

func (h *Host) Close() error {
	if h.cmd != nil {
		log.Debugf("%s: waiting for rpc worker...", h.name)
		return h.cmd.Wait()
	}

	return nil
}

func (h *Host) Functions() map[string]function.Function {
	return nil
}
