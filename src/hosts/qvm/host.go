package qvm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/illikainen/orch/src/embeds"
	"github.com/illikainen/orch/src/metadata"
	"github.com/illikainen/orch/src/qubes"
	"github.com/illikainen/orch/src/rpc/controller"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type Host struct {
	Condition bool
	Hostname  string
	Become    string
	name      string
	bin       string
	sys       *sysinfo
	value     cty.Value
	cmd       *exec.Cmd
	shutdown  bool
}

func (h *Host) Decode(name string, body hcl.Body, ctx *hcl.EvalContext) error {
	value, diags := hcldec.Decode(
		body,
		&hcldec.ObjectSpec{
			"condition": &hcldec.AttrSpec{
				Name: "condition",
				Type: cty.Bool,
			},
			"hostname": &hcldec.AttrSpec{
				Name: "hostname",
				Type: cty.String,
			},
			"become": &hcldec.AttrSpec{
				Name: "become",
				Type: cty.String,
			},
		},
		ctx,
	)
	if diags != nil {
		return diags
	}

	err := utils.FromCtyValue(value, h)
	if err != nil {
		return err
	}

	if value.GetAttr("condition").IsNull() {
		h.Condition = true
	}

	h.name = name
	if h.Hostname == "" {
		h.Hostname = name
	}

	h.value = value

	return nil
}

func (h *Host) Validate() error {
	return nil
}

func (h *Host) Include() bool {
	return h.Condition
}

func (h *Host) Value() cty.Value {
	return h.value
}

func (h *Host) Dial() error {
	log.Debugf("qvm: connecting to %s", h.Hostname)

	dom0, err := qubes.IsDom0()
	if err != nil {
		return err
	}

	if dom0 {
		vm, err := qubes.Find(h.Hostname)
		if err != nil {
			return err
		}

		if vm.State != qubes.RunningState {
			err := qubes.Start(h.Hostname)
			if err != nil {
				return err
			}

			h.shutdown = true
		}
	}

	info, err := h.getSysInfo()
	if err != nil {
		return err
	}
	log.Debugf("os=%s, arch=%s, home=%s", info.os, info.arch, info.home)
	h.sys = info
	h.bin = filepath.Join(info.home, ".cache", metadata.Name(), "bin", metadata.Name())

	return nil
}

type sysinfo struct {
	os   string
	arch string
	home string
}

func (h *Host) getSysInfo() (*sysinfo, error) {
	uname, err := qubes.Exec(&qubes.ExecOptions{
		Name:    h.Hostname,
		Command: []string{"uname", "-s", "-m"},
		Become:  h.Become,
	})
	if err != nil {
		return nil, err
	}

	elts := strings.Split(strings.TrimRight(string(uname.Stdout), "\n"), " ")
	if len(elts) != 2 {
		return nil, errors.Errorf("invalid output: %s", uname.Stdout)
	}

	arch := elts[1]
	if arch == "x86_64" {
		arch = "amd64"
	}

	printenv, err := qubes.Exec(&qubes.ExecOptions{
		Name:    h.Hostname,
		Command: []string{"printenv", "HOME"},
		Become:  h.Become,
	})
	if err != nil {
		return nil, err
	}
	home := strings.TrimRight(string(printenv.Stdout), "\n")

	return &sysinfo{
		os:   strings.ToLower(elts[0]),
		arch: arch,
		home: home,
	}, nil
}

func (h *Host) Name() string {
	return h.name
}

func (h *Host) UploadBinary() (err error) {
	name := fmt.Sprintf("%s_%s_%s", metadata.Name(), h.sys.os, h.sys.arch)
	f, err := embeds.OpenBin(name)
	if err != nil {
		return err
	}
	defer errorx.Defer(f.Close, &err)

	hsh := sha256.New()
	err = iofs.Copy(hsh, f)
	if err != nil {
		return err
	}

	fSeek, ok := f.(io.ReadSeeker)
	if !ok {
		return errors.Errorf("bug")
	}
	_, err = fSeek.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	cksum := hex.EncodeToString(hsh.Sum(nil))
	log.Tracef("%s: %s: sha256=%s", h.Hostname, name, cksum)

	out, err := qubes.Exec(&qubes.ExecOptions{
		Name:    h.Hostname,
		Command: []string{"sha256sum", "--", h.bin},
		Become:  h.Become,
	})
	if err == nil {
		elts := strings.Split(string(out.Stdout), " ")
		if len(elts) != 3 {
			return errors.Errorf("unexpected output length")
		}

		if elts[0] == cksum {
			log.Debugf("%s: using cached %s", h.Hostname, h.bin)
			return nil
		}
	}

	log.Infof("%s: uploading %s to %s", h.Hostname, name, h.bin)
	_, err = qubes.Exec(&qubes.ExecOptions{
		Name:    h.Hostname,
		Command: []string{"mkdir", "-p", "--", filepath.Dir(h.bin)},
		Become:  h.Become,
	})
	if err != nil {
		return err
	}

	_, err = qubes.Exec(&qubes.ExecOptions{
		Name:    h.Hostname,
		Command: []string{"tee", "--", h.bin},
		Become:  h.Become,
		Stdin:   f,
	})
	if err != nil {
		return err
	}

	_, err = qubes.Exec(&qubes.ExecOptions{
		Name:    h.Hostname,
		Command: []string{"chmod", "u+x", "--", h.bin},
		Become:  h.Become,
	})
	if err != nil {
		return err
	}

	return nil
}

func (h *Host) Start() (*controller.Controller, error) {
	args, err := qubes.Command(&qubes.CommandOptions{
		Name:    h.Hostname,
		Command: []string{h.bin, "_rpc"},
		Become:  h.Become,
	})
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(args[0], args[1:]...) // #nosec G204

	w, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	r, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	log.Tracef("exec: %s", strings.Join(cmd.Args, " "))
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	ctrl := controller.New(r, w)
	err = ctrl.Start()
	if err != nil {
		return nil, err
	}

	h.cmd = cmd
	return ctrl, nil
}

func (h *Host) Close() error {
	var errs []error

	if h.cmd != nil {
		log.Debugf("%s: waiting for rpc worker...", h.name)
		errs = append(errs, h.cmd.Wait())
	}

	if h.shutdown {
		errs = append(errs, qubes.Shutdown(h.Hostname, true))
	}

	return errorx.Join(errs...)
}

func (h *Host) Functions() map[string]function.Function {
	return nil
}
