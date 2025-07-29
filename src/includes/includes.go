package includes

import (
	"github.com/illikainen/orch/src/hclang"
)

type Includes []*Include

func (i *Includes) PartialDecode(basedir string) error {
	for _, include := range *i {
		err := include.PartialDecode(basedir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Includes) Decode(ctx *hclang.EvalContext) error {
	for _, include := range *i {
		err := include.Decode(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
