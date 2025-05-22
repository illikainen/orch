package blueprint

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func localFunctions() map[string]function.Function {
	return map[string]function.Function{
		"oct":   oct(),
		"print": printer(),
	}
}

func oct() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "num",
				Type: cty.Number,
			},
		},
		Type: function.StaticReturnType(cty.Number),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			num, accuracy := args[0].AsBigFloat().Int64()
			if accuracy != big.Exact {
				return cty.NilVal, errors.Errorf("invalid integer: %f", args[0].AsBigFloat())
			}

			conv, err := strconv.ParseInt(fmt.Sprintf("%d", num), 8, 64)
			if err != nil {
				return cty.NilVal, err
			}

			return cty.NumberIntVal(conv), nil
		},
	})
}

func printer() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "obj",
				Type: cty.DynamicPseudoType,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			data, err := ctyjson.Marshal(args[0], args[0].Type())
			if err != nil {
				return cty.StringVal(""), err
			}

			log.Infof("%s", data)
			return cty.StringVal(string(data)), nil
		},
	})
}
