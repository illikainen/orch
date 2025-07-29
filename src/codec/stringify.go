package codec

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type Stringify string

func (s *Stringify) UnmarshalJSON(data []byte) error {
	var str string

	err := json.Unmarshal(data, &str)
	if err == nil {
		*s = Stringify(str)
		return nil
	}

	var num json.Number
	err = json.Unmarshal(data, &num)
	if err == nil {
		*s = Stringify(num.String())
		return nil
	}

	return errors.Errorf("unsupported stringify JSON value: %s", data)
}
