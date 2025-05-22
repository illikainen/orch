package fact

import (
	"regexp"

	"github.com/illikainen/go-utils/src/iofs"
	"github.com/illikainen/go-utils/src/stringx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type OS struct {
	Name     string `cty:"name"`
	Version  string `cty:"version"`
	Codename string `cty:"codename"`
}

func GatherOSFacts() (*OS, error) {
	data, err := iofs.ReadFile("/etc/os-release")
	if err != nil {
		return nil, err
	}

	osRelease := OS{}

	rx, err := regexp.Compile("(^[A-Z_]+)=\"?([^\"$]*)\"?$")
	if err != nil {
		return nil, err
	}

	for _, line := range stringx.SplitLines(string(data)) {
		match := rx.FindStringSubmatch(line)
		if match == nil {
			return nil, errors.Errorf("unparseable os-release line: %s", line)
		}

		switch match[1] {
		case "ID":
			osRelease.Name = match[2]
		case "VERSION_ID":
			osRelease.Version = match[2]
		case "VERSION_CODENAME":
			osRelease.Codename = match[2]
		default:
			log.Debugf("skipping unknown os-release line: %s", line)
		}
	}

	return &osRelease, nil
}
