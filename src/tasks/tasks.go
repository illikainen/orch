package tasks

import (
	"github.com/hashicorp/hcl/v2"
)

type Tasks []*Task

func (t *Tasks) PartialDecode() error {
	for _, task := range *t {
		if err := task.PartialDecode(); err != nil {
			return err
		}
	}
	return nil
}

func (t *Tasks) Decode(_ func() (*hcl.EvalContext, error)) error {
	return nil
}

func (t *Tasks) Dependencies() []string {
	deps := []string{}
	for _, task := range *t {
		deps = append(deps, task.Dependencies...)
	}
	return deps
}
