package ssh

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/illikainen/orch/src/embeds"
	"github.com/illikainen/orch/src/metadata"
	"github.com/illikainen/orch/src/rpc/controller"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-netutils/src/sshx"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/crypto/ssh"
)

type Host struct {
	Condition bool
	Hostname  string
	User      string
	Password  string
	Become    string
	name      string
	conn      *sshx.Client
	bin       string
	sys       *sysinfo
	value     cty.Value
	session   *ssh.Session
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
			"user": &hcldec.AttrSpec{
				Name: "user",
				Type: cty.String,
			},
			"password": &hcldec.AttrSpec{
				Name: "password",
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

func (h *Host) Dial(_ bool) error {
	log.Debugf("ssh: connecting to %s", h.Hostname)

	conn, err := sshx.Dial("tcp", h.Hostname, &sshx.ClientConfig{
		User:     h.User,
		Password: h.Password,
	})
	if err != nil {
		return err
	}
	h.conn = conn

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

// We need to get this information to know which binary to upload to gather
// facts and execute tasks.
func (h *Host) getSysInfo() (*sysinfo, error) {
	uname, err := h.conn.Exec(&sshx.ExecOptions{
		Command: "uname -s -m",
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

	printenv, err := h.conn.Exec(&sshx.ExecOptions{
		Command: "printenv HOME",
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

	out, err := h.conn.Exec(&sshx.ExecOptions{
		Command: fmt.Sprintf("sha256sum -- '%s'", strings.ReplaceAll(h.bin, "'", "'\\''")),
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
	_, err = h.conn.Exec(&sshx.ExecOptions{
		Command: fmt.Sprintf("mkdir -p -- '%s'", strings.ReplaceAll(filepath.Dir(h.bin), "'", "'\\''")),
		Become:  h.Become,
	})
	if err != nil {
		return err
	}

	_, err = h.conn.Exec(&sshx.ExecOptions{
		Command: fmt.Sprintf("tee -- %s", strings.ReplaceAll(h.bin, "'", "'\\''")),
		Become:  h.Become,
		Stdin:   f,
	})
	if err != nil {
		return err
	}

	_, err = h.conn.Exec(&sshx.ExecOptions{
		Command: fmt.Sprintf("chmod +x -- '%s'", strings.ReplaceAll(h.bin, "'", "'\\''")),
		Become:  h.Become,
	})
	if err != nil {
		return err
	}

	return nil
}

func (h *Host) Start() (*controller.Controller, error) {
	session, err := h.conn.NewSession()
	if err != nil {
		return nil, err
	}

	w, err := session.StdinPipe()
	if err != nil {
		return nil, err
	}

	r, err := session.StderrPipe()
	if err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf("'%s' _rpc", strings.ReplaceAll(h.bin, "'", "'\\''"))
	if h.Become != "" {
		esc, err := h.conn.Become(h.Become)
		if err != nil {
			return nil, err
		}

		cmd = fmt.Sprintf("%s %s", esc, cmd)
	}

	log.Tracef("exec: %s", cmd)
	err = session.Start(cmd)
	if err != nil {
		return nil, err
	}
	h.session = session

	ctrl := controller.New(r, w)
	err = ctrl.Start()
	if err != nil {
		return nil, err
	}

	return ctrl, nil
}

func (h *Host) Close() error {
	errs := []error{}

	if h.session != nil {
		log.Debugf("%s: waiting for rpc worker...", h.name)
		errs = append(errs, errors.WithStack(h.session.Wait()))
	}

	if h.conn != nil {
		errs = append(errs, errors.WithStack(h.conn.Close()))
	}

	return errorx.Join(errs...)
}
