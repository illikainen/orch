package utils

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

type Uname struct {
	OS   string
	Arch string
}

func ParseUname(s string) (*Uname, error) {
	elts := strings.Split(strings.TrimRight(s, "\n"), " ")
	if len(elts) != 2 {
		return nil, errors.Errorf("invalid output for uname -s -m")
	}

	uname := &Uname{}

	switch elts[0] {
	case "Linux":
		uname.OS = "linux"
	case "Darwin":
		uname.OS = "darwin"
	default:
		return nil, errors.Errorf("invalid os: %s", elts[0])
	}

	switch elts[1] {
	case "aarch64", "arm64":
		uname.Arch = "arm64"
	case "x86_64":
		uname.Arch = "amd64"
	default:
		return nil, errors.Errorf("invalid arch: %s", elts[1])
	}

	return uname, nil
}

func ParseSHA256(s string) (string, error) {
	elts := strings.Split(s, " ")
	if len(elts) != 3 {
		return "", errors.Errorf("bad sha256sum output: %s", s)
	}

	match, err := regexp.Match(`^[0-9a-f]{64}$`, []byte(elts[0]))
	if err != nil {
		return "", errors.WithStack(err)
	}

	if !match {
		return "", errors.Errorf("bad sha256sum: %s", elts[0])
	}

	return elts[0], nil
}

func ParsePath(s string) (string, error) {
	path := strings.TrimRight(s, "\n")
	match, err := regexp.Match(`^[a-zA-Z0-9/,.]+$`, []byte(path))
	if err != nil {
		return "", errors.WithStack(err)
	}

	if !match || strings.Contains(path, "..") {
		return "", errors.Errorf("'%s' is not a valid path", path)
	}

	return path, nil
}
