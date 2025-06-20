package worker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/illikainen/go-utils/src/stringx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/illikainen/orch/src/rpc"
)

var workers = map[string]func([]byte) (any, error){}

func Register(name string, fun func(data []byte) (any, error)) error {
	if _, ok := workers[name]; ok {
		return errors.Errorf("%s is already registered", name)
	}

	workers[name] = fun
	return nil
}

type Worker struct {
	reader io.Reader
	writer io.Writer
	group  errgroup.Group
}

func New(r io.Reader, w io.Writer) *Worker {
	return &Worker{
		reader: r,
		writer: w,
	}
}

func (w *Worker) Start() error {
	w.group.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				err := w.Return(&rpc.Return{
					Error: errors.Errorf("%s", r),
					Fatal: true,
				})
				if err != nil {
					log.Errorf("%v", err)
				}

				os.Exit(1) // revive:disable-line:deep-exit
			}
		}()

		scan := bufio.NewScanner(w.reader)
		for scan.Scan() {
			data := scan.Bytes()
			if !bytes.Equal(data, stringx.Sanitize(data)) {
				return errors.Errorf("worker received invalid data")
			}

			var msg rpc.Message
			err := json.Unmarshal(data, &msg)
			if err != nil {
				return errors.WithStack(err)
			}
			log.Tracef("message: %s", data)

			switch msg.Type {
			case rpc.ControlType:
				var c rpc.Control
				err := json.Unmarshal(data, &c)
				if err != nil {
					e := w.Return(&rpc.Return{
						Error: errors.Wrap(err, "control"),
					})
					if e != nil {
						return e
					}
					continue
				}

				if c.State == rpc.ExitState {
					return nil
				}
			case rpc.FunctionCallType:
				var fc rpc.FunctionCall
				err := json.Unmarshal(data, &fc)
				if err != nil {
					e := w.Return(&rpc.Return{
						Error: errors.Wrap(err, "function call"),
					})
					if e != nil {
						return e
					}
					continue
				}

				fun, ok := workers[fc.Function]
				if !ok {
					err := w.Return(&rpc.Return{
						Error: errors.Errorf("invalid function: %s", fc.Function),
					})
					if err != nil {
						return err
					}
					continue
				}

				rv, err := fun(fc.Data)
				if err != nil {
					e := w.Return(&rpc.Return{
						Error: err,
					})
					if e != nil {
						return err
					}
					continue
				}

				data, err := json.Marshal(rv)
				if err != nil {
					e := w.Return(&rpc.Return{
						Error: err,
					})
					if e != nil {
						return e
					}
					continue
				}

				err = w.Return(&rpc.Return{
					Value: data,
				})
				if err != nil {
					return err
				}
			default:
				err := w.Return(&rpc.Return{
					Error: errors.Errorf("worker received invalid type %d", msg.Type),
				})
				if err != nil {
					return err
				}
			}
		}
		return errors.WithStack(scan.Err())
	})

	return nil
}

func (w *Worker) Return(rv *rpc.Return) error {
	ret := *rv
	ret.Type = rpc.ReturnType

	data, err := json.Marshal(&ret)
	if err != nil {
		return errors.WithStack(err)
	}
	data = append(data, '\n')

	if !bytes.Equal(data, stringx.Sanitize(data)) {
		return errors.Errorf("worker received invalid return data")
	}

	n, err := w.writer.Write(data)
	if err != nil {
		return errors.WithStack(err)
	}
	if n != len(data) {
		return errors.Errorf("invalid write size")
	}

	return nil
}

func (w *Worker) Wait() error {
	return errors.WithStack(w.group.Wait())
}
