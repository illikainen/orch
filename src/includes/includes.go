package includes

import (
	"github.com/hashicorp/hcl/v2"
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

func (i *Includes) Decode(ctxfn func() (*hcl.EvalContext, error)) error {
	for _, include := range *i {
		err := include.Decode(ctxfn)
		if err != nil {
			return err
		}
	}
	return nil
}
