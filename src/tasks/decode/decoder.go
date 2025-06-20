package decode

import (
	"github.com/illikainen/orch/src/configs"

	"github.com/hashicorp/hcl/v2"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type Decoder interface {
	Decode(hcl.Body, *hcl.EvalContext, *configs.Config) error
	Validate() error
	Include() bool
	Value() cty.Value
}

var decoders = map[string]func() (Decoder, error){}

func Register(name string, fun func() (Decoder, error)) error {
	if _, ok := decoders[name]; ok {
		return errors.Errorf("%s is already registered", name)
	}

	decoders[name] = fun
	return nil
}

func Lookup(name string) (Decoder, error) {
	fun, ok := decoders[name]
	if !ok {
		return nil, errors.Errorf("%s is not a valid decoder", name)
	}

	return fun()
}
