package codec

import (
	"encoding/json"
	"regexp"
)

type Regexp struct {
	*regexp.Regexp
}

func (r *Regexp) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *Regexp) UnmarshalJSON(data []byte) error {
	var expr string

	err := json.Unmarshal(data, &expr)
	if err != nil {
		return err
	}

	rx, err := regexp.Compile(expr)
	if err != nil {
		return err
	}

	r.Regexp = rx
	return nil
}
