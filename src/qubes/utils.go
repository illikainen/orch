package qubes

import (
	"github.com/illikainen/go-utils/src/iofs"
)

func IsDom0() (bool, error) {
	dir, err := iofs.Exists("/var/run/qubes")
	if err != nil {
		return false, err
	}

	agent, err := iofs.Exists("/var/run/qubes/qrexec-agent")
	if err != nil {
		return false, err
	}

	return dir && !agent, nil
}

func IsQVM() (bool, error) {
	dir, err := iofs.Exists("/var/run/qubes")
	if err != nil {
		return false, err
	}

	agent, err := iofs.Exists("/var/run/qubes/qrexec-agent")
	if err != nil {
		return false, err
	}

	return dir && agent, nil
}
