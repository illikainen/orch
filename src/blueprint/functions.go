package blueprint

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

type evalContextFunctions struct{}

func (e *evalContextFunctions) Functions() (map[string]function.Function, error) {
	return map[string]function.Function{
		"getattr": getattr(),
		"oct":     oct(),
		"print":   printer(),
	}, nil
}

func getattr() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "values",
				Type: cty.Map(cty.DynamicPseudoType),
			},
			{
				Name: "key",
				Type: cty.String,
			},
			{
				Name: "fallback",
				Type: cty.DynamicPseudoType,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			return args[2].Type(), nil
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			values := args[0].AsValueMap()
			key := args[1].AsString()
			fallback := args[2]

			if value, ok := values[key]; ok {
				valType := value.Type()
				if valType.Equals(retType) {
					return value, nil
				}

				if valType.Equals(cty.String) {
					str := value.AsString()

					if retType.Equals(cty.Bool) {
						v, err := strconv.ParseBool(str)
						if err != nil {
							return cty.NilVal, err
						}
						return cty.BoolVal(v), nil
					}

					if retType.Equals(cty.Number) {
						v, err := strconv.ParseFloat(str, reflect.TypeOf(float64(0)).Bits())
						if err != nil {
							return cty.NilVal, err
						}
						return cty.NumberFloatVal(v), nil
					}
				}

				return cty.NilVal, errors.Errorf("getattr: unsupported type: %s", valType)
			}

			return fallback, nil
		},
	})
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
