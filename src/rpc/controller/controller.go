package controller

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"

	"github.com/illikainen/orch/src/rpc"

	"github.com/illikainen/go-utils/src/logging"
	"github.com/illikainen/go-utils/src/stringx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Controller struct {
	Log     logging.Logger
	reader  io.Reader
	writer  io.Writer
	returns chan *rpc.Return
	group   errgroup.Group
	fatal   bool
}

func New(r io.Reader, w io.Writer) *Controller {
	return &Controller{
		Log:     log.StandardLogger(),
		reader:  r,
		writer:  w,
		returns: make(chan *rpc.Return),
	}
}

func (c *Controller) Call(opts *rpc.FunctionCall) (json.RawMessage, error) {
	params, err := json.Marshal(opts.Params)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	call := *opts
	call.Type = rpc.FunctionCallType
	call.Params = params

	data, err := json.Marshal(call)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	data = append(data, '\n')

	n, err := c.writer.Write(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if n != len(data) {
		return nil, errors.Errorf("invalid write size")
	}

	rv := <-c.returns
	if rv.Error != nil {
		return nil, rv.Error
	}

	return rv.Value, nil
}

func (c *Controller) Start() error {
	c.group.Go(func() error {
		scan := bufio.NewScanner(c.reader)
		for scan.Scan() {
			data := scan.Bytes()
			if !bytes.Equal(data, stringx.Sanitize(data)) {
				return errors.Errorf("controller received invalid data")
			}

			var msg rpc.Message
			err := json.Unmarshal(data, &msg)
			if err != nil {
				return errors.WithStack(err)
			}

			switch msg.Type {
			case rpc.LogType:
				var l rpc.Log
				err := json.Unmarshal(data, &l)
				if err != nil {
					return errors.WithStack(err)
				}

				var fields log.Fields
				err = json.Unmarshal([]byte(l.Fields), &fields)
				if err != nil {
					return errors.WithStack(err)
				}

				level, err := log.ParseLevel(logging.GetField(fields, "level", "info"))
				if err != nil {
					return errors.WithStack(err)
				}

				log.WithFields(fields).Logln(level, "worker: "+logging.GetField(fields, "msg", "n/a"))
			case rpc.ReturnType:
				var rv rpc.Return
				err := json.Unmarshal(data, &rv)
				if err != nil {
					return errors.WithStack(err)
				}

				c.returns <- &rv

				if rv.Fatal {
					c.fatal = true
					return rv.Error
				}
			}
		}
		return scan.Err()
	})

	return nil
}

func (c *Controller) Close() error {
	if !c.fatal {
		data, err := json.Marshal(rpc.Control{
			Type:  rpc.ControlType,
			State: rpc.ExitState,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		data = append(data, '\n')

		n, err := c.writer.Write(data)
		if err != nil {
			return errors.WithStack(err)
		}
		if n != len(data) {
			return errors.Errorf("invalid write size")
		}
	}

	return c.group.Wait()
}
