package roles

import (
	"github.com/illikainen/go-utils/src/seq"
	"github.com/pkg/errors"

	"github.com/illikainen/orch/src/hclang"
)

type Roles []*Role

func (r *Roles) PartialDecode() error {
	for _, role := range *r {
		err := role.PartialDecode()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Roles) Decode(ctx *hclang.EvalContext) error {
	for _, role := range *r {
		if err := role.Decode(ctx); err != nil {
			return err
		}
	}
	return r.Validate()
}

func (r *Roles) Dependencies() []string {
	deps := []string{}
	for _, role := range *r {
		deps = append(deps, role.Dependencies...)
	}
	return deps
}

func (r *Roles) Validate() error {
	seen := []string{}
	for _, role := range *r {
		if err := role.Validate(); err != nil {
			return err
		}

		if seq.Contains(seen, role.Unique()) {
			return errors.Errorf("\"%s\" is not unique", role.Unique())
		}
		seen = append(seen, role.Unique())
	}
	return nil
}
