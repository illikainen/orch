package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func ToCtyValue(value any) (cty.Value, error) {
	typ, err := gocty.ImpliedType(value)
	if err != nil {
		return cty.NilVal, err
	}

	return gocty.ToCtyValue(value, typ)
}

// Ugly workaround because gocty.FromCtyValue() doesn't support optional values.
func FromCtyValue(value cty.Value, out any) error {
	data, err := ctyjson.Marshal(value, value.Type())
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, out)
	if err != nil {
		return err
	}

	return nil
}

func MergeCtyValues(a map[string]cty.Value, b map[string]cty.Value) (map[string]cty.Value, error) {
	result := map[string]cty.Value{}

	for aKey, aValue := range a {
		result[aKey] = aValue
	}

	for bKey, bValue := range b {
		if aValue, ok := result[bKey]; ok {
			if aValue.Type() != cty.Map(cty.String) || bValue.Type() != cty.Map(cty.String) {
				return nil, errors.Errorf("duplicate values for identifier `%s'", bKey)
			}

			aMap := aValue.AsValueMap()
			bMap := bValue.AsValueMap()

			rMap, err := MergeCtyValues(aMap, bMap)
			if err != nil {
				return nil, err
			}
			result[bKey] = cty.ObjectVal(rMap)
		} else {
			result[bKey] = bValue
		}
	}

	return result, nil
}

func JoinCtyPath[T *hclsyntax.Body | string](base T, sub string) (string, error) {
	var basedir string

	switch b := any(base).(type) {
	case *hclsyntax.Body:
		basedir = filepath.Dir(b.SrcRange.Filename)
	case string:
		basedir = b
	default:
		return "", errors.Errorf("invalid type for %v", base)
	}

	basedir, err := filepath.Abs(basedir)
	if err != nil {
		return "", err
	}
	basedir = filepath.Clean(basedir)

	path, err := filepath.Abs(filepath.Join(basedir, sub))
	if err != nil {
		return "", err
	}
	path = filepath.Clean(path)

	if !strings.HasPrefix(path+string(os.PathSeparator), basedir+string(os.PathSeparator)) {
		return "", errors.Errorf("%s is not a valid path", path)
	}

	return path, nil
}
