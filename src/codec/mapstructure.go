package codec

import (
	"reflect"
	"strconv"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

func StringToPrimitiveHookFunc() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if from.Kind() == reflect.String && to.Kind() == reflect.Ptr {
			s, ok := data.(string)
			if !ok {
				return nil, errors.Errorf("bug")
			}

			switch to.Elem().Kind() {
			case reflect.Bool:
				v, err := strconv.ParseBool(s)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				return &v, nil
			case reflect.Int64:
				v, err := strconv.ParseInt(s, 10, reflect.TypeOf(int64(0)).Bits())
				if err != nil {
					return nil, errors.WithStack(err)
				}
				return &v, nil
			}
		}
		return data, nil
	}
}
