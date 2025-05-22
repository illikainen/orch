package includes

import (
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
)

type Include struct {
	Name string `hcl:"name,label"`
	Src  string `hcl:"src"`
}

func (i *Include) PartialDecode(basedir string) error {
	src, err := utils.JoinCtyPath(basedir, i.Src)
	if err != nil {
		return err
	}

	i.Src = src
	return nil
}

func (i *Include) Decode(_ func() (*hcl.EvalContext, error)) error {
	return nil
}
