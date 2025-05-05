package embeds

import (
	"io/fs"
	"path"

	log "github.com/sirupsen/logrus"
)

func OpenBin(name string) (fs.File, error) {
	log.Debugf("embeds: opening binary %s", name)
	return bin.Open(path.Join("files", "bin", name))
}
