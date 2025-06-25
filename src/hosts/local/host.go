package local

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/process"
	"github.com/pkg/errors"

	"github.com/illikainen/orch/src/embeds"
	"github.com/illikainen/orch/src/metadata"
	"github.com/illikainen/orch/src/rpc/controller"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
)

type Host struct {
	Condition bool
	Hostname  string
	Become    string
	name      string
	bin       string
	value     cty.Value
	cmd       *exec.Cmd
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

	h.bin = filepath.Join(filepath.Join(".cache", metadata.Name()), "bin", metadata.Name())
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

func (h *Host) Dial(_ bool) error {
	log.Debugf("local: connecting to %s", h.Hostname)

	var usr *user.User
	if h.Become == "" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		usr = u
	} else {
		u, err := user.Lookup(h.Become)
		if err != nil {
			return err
		}
		usr = u
	}

	h.bin = filepath.Join(usr.HomeDir, ".cache", metadata.Name(), "bin", metadata.Name())

	return nil
}

func (h *Host) Name() string {
	return h.name
}

func (h *Host) UploadBinary() (err error) {
	name := fmt.Sprintf("%s_%s_%s", metadata.Name(), runtime.GOOS, runtime.GOARCH)
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

	out, err := process.Exec(&process.ExecOptions{
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
	_, err = process.Exec(&process.ExecOptions{
		Command: []string{"mkdir", "-p", "--", filepath.Dir(h.bin)},
		Become:  h.Become,
	})
	if err != nil {
		return err
	}

	_, err = process.Exec(&process.ExecOptions{
		Command: []string{"tee", "--", h.bin},
		Become:  h.Become,
		Stdin:   f,
	})
	if err != nil {
		return err
	}

	_, err = process.Exec(&process.ExecOptions{
		Command: []string{"chmod", "u+x", "--", h.bin},
		Become:  h.Become,
	})
	if err != nil {
		return err
	}

	return nil
}

func (h *Host) Start() (*controller.Controller, error) {
	args := []string{h.bin, "_rpc"}

	if h.Become != "" {
		esc, err := process.Become(h.Become)
		if err != nil {
			return nil, err
		}
		args = append(esc, args...)
	}

	log.Tracef("exec: %s", strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...) // #nosec G204
	w, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	r, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

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
	if h.cmd != nil {
		log.Debugf("%s: waiting for rpc worker...", h.name)
		return h.cmd.Wait()
	}

	return nil
}
