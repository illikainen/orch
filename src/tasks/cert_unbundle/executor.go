//lint:ignore ST1003 readability
package cert_unbundle // revive:disable-line:var-naming

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/illikainen/orch/src/rpc/worker"
	"github.com/illikainen/orch/src/tasks/file_manage"
	"github.com/illikainen/orch/src/tasks/file_remove"
	"github.com/illikainen/orch/src/tasks/outputs"

	"github.com/illikainen/go-utils/src/fn"
	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/seq"
	"github.com/illikainen/go-utils/src/stringx"
	"github.com/pkg/errors"
)

func init() {
	fn.Must(worker.Register("cert_unbundle", NewExecutor))
}

type Executor struct {
	Task
}

func NewExecutor() (worker.Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) Execute() (any, error) {
	data, err := iofs.ReadFile(e.Src)
	if err != nil {
		return nil, err
	}

	var changes []string
	var seen []string

	for len(data) != 0 {
		block, rest := pem.Decode(data)
		if block == nil || block.Type != "CERTIFICATE" || bytes.Equal(rest, data) {
			return nil, errors.Errorf("invalid certificate")
		}
		data = rest

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if !cert.IsCA {
			return nil, errors.Errorf("%s is not a CA", cert.Subject.CommonName)
		}

		name, err := basename(cert)
		if err != nil {
			return nil, err
		}

		if !seq.Contains(seen, name) {
			certData := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: block.Bytes,
			})

			mkdirChanges, err := file_manage.Mkdir(e.Dst, e.DirMode, e.Config.DryRun)
			if err != nil {
				return nil, err
			}
			changes = append(changes, mkdirChanges...)

			writeChanges, err := file_manage.WriteFile(
				filepath.Join(e.Dst, name),
				certData,
				e.FileMode,
				e.Config.DryRun,
			)
			if err != nil {
				return nil, err
			}
			changes = append(changes, writeChanges...)
			seen = append(seen, name)
		}
	}

	elts, err := os.ReadDir(e.Dst)
	if err != nil {
		if !e.Config.DryRun || !errors.Is(err, os.ErrNotExist) {
			return nil, errors.WithStack(err)
		}
		elts = nil
	}

	for _, elt := range elts {
		if !seq.Contains(seen, elt.Name()) {
			removeChanges, err := file_remove.Remove(filepath.Join(e.Dst, elt.Name()), e.Config.DryRun)
			if err != nil {
				return nil, err
			}
			changes = append(changes, removeChanges...)
		}
	}

	return &outputs.Output{
		Changed: changes != nil,
		Diff: map[string][]string{
			"certs": changes,
		},
	}, nil
}

func basename(cert *x509.Certificate) (string, error) {
	name := cert.Subject.CommonName
	if name == "" {
		if len(cert.Subject.OrganizationalUnit) != 1 {
			return "", errors.Errorf("CA unknown name")
		}
		name = cert.Subject.OrganizationalUnit[0]
	}

	name = stringx.Sanitize(strings.ReplaceAll(name, " ", "_"))
	match, err := regexp.Match(`^[a-zA-Z0-9()._-]+$`, []byte(name))
	if err != nil {
		return "", errors.WithStack(err)
	}

	if !match || strings.Contains(name, "..") {
		return "", errors.Errorf("'%s' is not a valid certificate name", name)
	}

	return name + ".pem", nil
}
