package includes

import (
	"github.com/illikainen/orch/src/hclang"
)

type Include struct {
	Name string `hcl:"name,label"`
	Src  string `hcl:"src"`
}

func (i *Include) PartialDecode(basedir string) error {
	src, err := hclang.JoinCtyPath(basedir, i.Src)
	if err != nil {
		return err
	}

	i.Src = src
	return nil
}

func (i *Include) Decode(_ *hclang.EvalContext) error {
	return nil
}
