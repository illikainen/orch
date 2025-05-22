package bindings

import (
	"path/filepath"

	"github.com/illikainen/orch/src/hosts"
	"github.com/illikainen/orch/src/roles"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/samber/lo"
	"github.com/zclconf/go-cty/cty"
)

type Binding struct {
	Name         string   `hcl:"name,label"`
	Hosts        []string `hcl:"hosts,optional"`
	Tags         []string `hcl:"tags,optional"`
	RoleDirs     []string `hcl:"roles,optional"`
	Roles        roles.Roles
	Dependencies []string
	value        cty.Value
}

func (b *Binding) PartialDecode(basedir string) error {
	for _, roledir := range b.RoleDirs {
		dir, err := utils.JoinCtyPath(basedir, roledir)
		if err != nil {
			return err
		}

		b.Roles = append(b.Roles, &roles.Role{
			Name: filepath.Base(dir),
			Dir:  dir,
		})
	}

	if err := b.Roles.PartialDecode(); err != nil {
		return err
	}

	for _, role := range b.Roles {
		b.Dependencies = role.Dependencies
	}

	return nil
}

func (b *Binding) Decode(ctxfn func() (*hcl.EvalContext, error)) error {
	return b.Roles.Decode(ctxfn)
}

func (b *Binding) Match(host *hosts.Host) bool {
	if lo.Contains(b.Hosts, host.Name) {
		return true
	}

	if len(lo.Intersect(b.Tags, host.Tags)) > 0 {
		return true
	}

	return false
}

func (b *Binding) Value() cty.Value {
	return b.value
}

func (b *Binding) Type() string {
	return "binding"
}

func (b *Binding) Unique() string {
	return b.Name
}

func (b *Binding) Validate() error {
	return nil
}
