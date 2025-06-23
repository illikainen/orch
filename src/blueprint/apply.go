package blueprint

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/illikainen/orch/src/qubes"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-netutils/src/sshx"
	"github.com/illikainen/go-utils/src/sandbox"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func Apply(opts *Options) error {
	output := outputs.Outputs{}

	// Apply local changes first in case localhost needs to be hardened before
	// communicating with remotes.
	if !sandbox.IsSandboxed() {
		var err error
		output, err = applyLocal(opts)
		if err != nil {
			return err
		}
	}

	// Re-execute ourselves in a sandbox on compatible systems before applying
	// on the remotes.
	if sandbox.Compatible() && !sandbox.IsSandboxed() {
		return startSandbox(output, opts)
	}

	return applyRemote(output, opts)
}

func applyLocal(opts *Options) (outputs.Outputs, error) {
	blueprint := NewBlueprint(opts)
	if err := blueprint.PartialDecode(); err != nil {
		return nil, err
	}

	output := outputs.Outputs{}
	for _, host := range blueprint.Hosts {
		if host.Type == "local" {
			out, err := blueprint.Apply(host.Name, output)
			if err != nil {
				return nil, err
			}
			output = append(output, out...)
		}
	}

	return output, nil
}

type worker struct {
	name   string
	output outputs.Outputs
	err    error
}

func applyRemote(output outputs.Outputs, opts *Options) error {
	// The output from non-sandboxed local applies is sent as JSON on stdin
	// to sandboxed subprocesses.
	if sandbox.IsSandboxed() {
		data := bytes.Buffer{}
		_, err := io.Copy(&data, os.Stdin)
		if err != nil {
			return err
		}

		err = json.Unmarshal(data.Bytes(), &output)
		if err != nil {
			return err
		}
	}

	blueprint := NewBlueprint(opts)
	if err := blueprint.PartialDecode(); err != nil {
		return err
	}

	channels := make([]chan worker, len(blueprint.Hosts))
	for i := range blueprint.Hosts {
		channels[i] = make(chan worker, len(blueprint.Hosts))
	}

	deps := blueprint.Dependencies.Filter(output.Hosts())
	if circular, host := deps.FindCircularDependencies(); circular {
		return errors.Errorf("circular dependency in %s", host)
	}

	group := errgroup.Group{}
	for i, host := range blueprint.Hosts {
		if host.Type == "local" {
			continue
		}

		idx := i
		name := host.Name
		out, err := output.Clone()
		if err != nil {
			return err
		}

		group.Go(func() error {
			bp := NewBlueprint(opts)
			if err := bp.PartialDecode(); err != nil {
				for _, c := range channels {
					c <- worker{name: name, err: err}
				}
				return err
			}

			for len(deps[name]) != 0 {
				log.Infof("%s: waiting for %s...", name, strings.Join(deps[name], ", "))

				done := <-channels[idx]
				deps = deps.Filter([]string{done.name})

				if done.err != nil {
					for _, c := range channels {
						c <- worker{name: name, err: done.err}
					}
					return done.err
				}

				out = append(out, done.output...)
			}

			newOut, err := bp.Apply(name, out)
			if err != nil {
				for _, c := range channels {
					c <- worker{name: name, err: err}
				}
				return err
			}

			for _, c := range channels {
				c <- worker{name: name, output: newOut, err: nil}
			}
			return nil
		})
	}

	return group.Wait()
}

func startSandbox(output outputs.Outputs, opts *Options) error {
	blueprint := NewBlueprint(opts)
	if err := blueprint.PartialDecode(); err != nil {
		return err
	}

	ro := []string{opts.Path}
	rw := []string{}
	dev := []string{}

	for _, include := range blueprint.Includes {
		ro = append(ro, include.Src)
	}

	for _, binding := range blueprint.Bindings {
		for _, role := range binding.Roles {
			ro = append(ro, role.Dir)
		}
	}

	sshRO, sshRW, err := sshx.SandboxPaths()
	if err != nil {
		return err
	}
	ro = append(ro, sshRO...)
	rw = append(rw, sshRW...)

	qvmRO, qvmRW, qvmDev, err := qubes.SandboxPaths()
	if err != nil {
		return err
	}
	ro = append(ro, qvmRO...)
	rw = append(rw, qvmRW...)
	dev = append(dev, qvmDev...)

	err = opts.Sandbox.AddReadOnlyPath(ro...)
	if err != nil {
		return err
	}

	err = opts.Sandbox.AddReadWritePath(rw...)
	if err != nil {
		return err
	}

	err = opts.Sandbox.AddDevPath(dev...)
	if err != nil {
		return err
	}

	data, err := json.Marshal(&output)
	if err != nil {
		return err
	}
	opts.Sandbox.SetStdin(bytes.NewReader(data))

	opts.Sandbox.SetShareNet(true)

	return opts.Sandbox.Confine()
}
