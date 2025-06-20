package rpc

import (
	"encoding/json"

	"github.com/pkg/errors"
)

const (
	ControlType = iota
	FunctionCallType
	LogType
	ReturnType
)

const (
	ExitState = iota
)

type Message struct {
	Type int
}

type Control struct {
	Type  int
	State int
}

type FunctionCall struct {
	Type     int
	Function string
	Data     json.RawMessage
}

type Log struct {
	Type   int
	Fields string
}

type Return struct {
	Type  int
	Value json.RawMessage
	Error error
	Fatal bool
}

func (r *Return) MarshalJSON() ([]byte, error) {
	type alias Return
	type retval struct {
		alias
		Error string
	}

	errstr := ""
	if r.Error != nil {
		errstr = r.Error.Error()
	}

	data, err := json.Marshal(retval{
		alias: alias(*r),
		Error: errstr,
	})

	return data, err
}

func (r *Return) UnmarshalJSON(data []byte) error {
	type alias Return
	type retval struct {
		alias
		Error string
	}
	var rv retval

	err := json.Unmarshal(data, &rv)
	if err != nil {
		return err
	}

	*r = Return(rv.alias)
	if rv.Error != "" {
		r.Error = errors.Errorf("%s", rv.Error)
	}
	return nil
}
