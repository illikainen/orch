package qvm

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/illikainen/orch/src/embeds"
	"github.com/illikainen/orch/src/fact"
	"github.com/illikainen/orch/src/metadata"
	"github.com/illikainen/orch/src/tasks"
	"github.com/illikainen/orch/src/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/illikainen/go-utils/src/errorx"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/process"
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
	os        string
	arch      string
	value     cty.Value
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

func (h *Host) Dial() error {
	log.Debugf("qvm: connecting to %s", h.Hostname)

	goos, goarch, err := h.getInfo()
	if err != nil {
		return err
	}
	log.Debugf("os=%s, arch=%s", goos, goarch)
	h.os = goos
	h.arch = goarch

	return nil
}

func (h *Host) getInfo() (os string, arch string, err error) {
	out, err := Exec(&ExecOptions{
		Name:    h.Hostname,
		Command: []string{"uname", "-s", "-m"},
	})
	if err != nil {
		return "", "", err
	}

	elts := strings.Split(strings.TrimRight(string(out.Stdout), "\n"), " ")
	if len(elts) != 2 {
		return "", "", errors.Errorf("invalid output: %s", out.Stdout)
	}

	arch = elts[1]
	if arch == "x86_64" {
		arch = "amd64"
	}

	return strings.ToLower(elts[0]), arch, nil
}

func (h *Host) Name() string {
	return h.name
}

func (h *Host) Close() error {
	return nil
}

func (h *Host) UploadBinary() (err error) {
	name := fmt.Sprintf("%s_%s_%s", metadata.Name(), h.os, h.arch)
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

	out, err := Exec(&ExecOptions{
		Name:    h.Hostname,
		Command: []string{"sha256sum", "--", h.bin},
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
	_, err = Exec(&ExecOptions{
		Name:    h.Hostname,
		Command: []string{"umask u=rwx,g=,o= && mkdir -p -- " + filepath.Dir(h.bin)},
	})
	if err != nil {
		return err
	}

	_, err = Exec(&ExecOptions{
		Name:    h.Hostname,
		Command: []string{"umask u=rwx,g=,o= && tee -- " + h.bin},
		Stdin:   f,
	})
	if err != nil {
		return err
	}

	_, err = Exec(&ExecOptions{
		Name:    h.Hostname,
		Command: []string{"chmod", "u+x", "--", h.bin},
	})
	if err != nil {
		return err
	}

	return nil
}

func (h *Host) GatherFacts() (*fact.Facts, error) {
	out, err := Exec(&ExecOptions{
		Name:    h.Hostname,
		Command: []string{h.bin, "_gather-facts"},
	})
	if err != nil {
		return nil, err
	}

	facts := &fact.Facts{}
	err = json.Unmarshal(out.Stdout, facts)
	if err != nil {
		return nil, err
	}

	return facts, nil
}

func (h *Host) Functions() map[string]function.Function {
	return nil
}

func (h *Host) Apply(task *tasks.Task) (tasks.Outputter, error) {
	data, err := json.MarshalIndent(task, "", "    ")
	if err != nil {
		return nil, err
	}
	log.Tracef("qvm: apply %s", data)

	out, err := Exec(&ExecOptions{
		Name:    h.Hostname,
		Command: []string{h.bin, "_apply-task"},
		Stdin:   bytes.NewReader(data),
		Stderr:  process.LogrusOutput,
	})
	if err != nil {
		return nil, err
	}

	output := &tasks.Output{}
	err = json.Unmarshal(out.Stdout, output)
	if err != nil {
		return nil, err
	}

	return output, nil
}
