package hosts

import (
	"github.com/illikainen/orch/src/hosts/local"
	"github.com/illikainen/orch/src/hosts/qvm"
	"github.com/illikainen/orch/src/hosts/ssh"
	"github.com/illikainen/orch/src/rpc/controller"

	"github.com/hashicorp/hcl/v2"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type Connector interface {
	Name() string
	Decode(string, hcl.Body, *hcl.EvalContext) error
	Validate() error
	Include() bool
	Value() cty.Value
	Dial() error
	UploadBinary() error
	Start() (*controller.Controller, error)
	Close() error
	Functions() map[string]function.Function
}

type Host struct {
	Type         string   `hcl:"type,label"`
	Name         string   `hcl:"host,label"`
	Tags         []string `hcl:"tags,optional"`
	Body         hcl.Body `hcl:"body,remain"`
	Dependencies []string
	Connector    Connector
}

func (h *Host) PartialDecode() error {
	attrs, diags := h.Body.JustAttributes()
	if diags != nil {
		return diags
	}

	for _, attr := range attrs {
		for _, v := range attr.Expr.Variables() {
			if len(v) >= 2 {
				if root, ok := v[0].(hcl.TraverseRoot); ok && root.Name == "out" {
					if host, ok := v[1].(hcl.TraverseAttr); ok && host.Name != "this" {
						h.Dependencies = append(h.Dependencies, host.Name)
					}
				}
			}
		}
	}

	return h.Validate()
}

func (h *Host) Decode(ctxfn func() (*hcl.EvalContext, error)) error {
	ctx, err := ctxfn()
	if err != nil {
		return err
	}

	connector, err := h.getConnector()
	if err != nil {
		return err
	}

	err = connector.Decode(h.Name, h.Body, ctx)
	if err != nil {
		return err
	}

	h.Connector = connector
	return connector.Validate()
}

func (h *Host) Validate() error {
	if h.Name == "this" {
		return errors.Errorf("`this' is a reserved name")
	}
	return nil
}

func (h *Host) Include() bool {
	return h.Connector.Include()
}

func (h *Host) Value() cty.Value {
	if h.Connector != nil {
		return h.Connector.Value()
	}
	return cty.NilVal
}

func (h *Host) Unique() string {
	return h.Name
}

func (h *Host) getConnector() (Connector, error) {
	switch h.Type {
	case "local":
		return &local.Host{}, nil
	case "qvm":
		return &qvm.Host{}, nil
	case "ssh":
		return &ssh.Host{}, nil
	}
	return nil, errors.Errorf("%s is not a valid host type for %s", h.Type, h.Name)
}
