package worker

import (
	"github.com/pkg/errors"
)

type Executor interface {
	Execute() (any, error)
}

var executors = map[string]func() (Executor, error){}

func Register(name string, fun func() (Executor, error)) error {
	if _, ok := executors[name]; ok {
		return errors.Errorf("%s is already registered", name)
	}

	executors[name] = fun
	return nil
}

func Lookup(name string) (Executor, error) {
	fun, ok := executors[name]
	if !ok {
		return nil, errors.Errorf("%s is not a valid executor", name)
	}

	return fun()
}
