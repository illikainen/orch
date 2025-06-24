package qubes

import (
	"github.com/illikainen/go-utils/src/process"
	log "github.com/sirupsen/logrus"
)

func Start(name string) error {
	log.Debugf("qubes: starting %s", name)
	_, err := process.Exec(&process.ExecOptions{
		Command: []string{"qvm-start", "--", name},
	})
	return err
}

func Shutdown(name string, wait bool) error {
	args := []string{"qvm-shutdown"}
	if wait {
		args = append(args, "--wait")
	}
	args = append(args, "--", name)

	log.Debugf("qubes: shutting down %s", name)
	_, err := process.Exec(&process.ExecOptions{
		Command: args,
	})
	return err
}
