//go:build generate

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/illikainen/go-utils/src/errorx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := writeMetadata("src/metadata/metadata.json")
	if err != nil {
		log.Fatalf("%s", err)
	}

	err = buildEmbeds("src/embeds/files/bin")
	if err != nil {
		log.Fatalf("%s", err)
	}
}

type metadata struct {
	Name    string
	Version string
	Commit  string
	Branch  string
}

func writeMetadata(file string) (err error) {
	commitCmd := exec.Command("git", "rev-parse", "HEAD") // #nosec G204
	commit, err := commitCmd.Output()
	if err != nil {
		return err
	}

	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD") // #nosec G204
	branch, err := branchCmd.Output()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(metadata{
		Name:    "orch",
		Version: "0.0.0",
		Commit:  strings.Trim(string(commit), "\r\n"),
		Branch:  strings.Trim(string(branch), "\r\n"),
	}, "", "    ")
	if err != nil {
		return err
	}

	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer errorx.Defer(f.Close, &err)

	data = append(data, '\n')
	n, err := f.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return errors.Errorf("invalid write")
	}

	return nil
}

type target struct {
	os   string
	arch string
}

func buildEmbeds(dir string) error {
	targets := []target{
		{os: runtime.GOOS, arch: runtime.GOARCH},
	}

	if os.Getenv("GOFER_RELEASE") == "true" {
		targets = []target{
			{os: "linux", arch: "amd64"},
			{os: "linux", arch: "arm64"},
			{os: "darwin", arch: "arm64"},
			{os: "windows", arch: "amd64"},
		}
	}

	for _, t := range targets {
		err := buildSelf(".", dir, t.os, t.arch)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildSelf(in string, out string, goos string, goarch string) error {
	name := fmt.Sprintf("orch_%s_%s", goos, goarch)
	outFile := filepath.Join(out, name)

	args := []string{
		"go",
		"build",
		"-mod=readonly",
		"-trimpath",
		"-buildmode=pie",
		"-ldflags",
		"-s -w -buildid=",
		"-tags=noembeds",
		"-o",
		outFile,
		in,
	}
	log.Info(strings.Join(args, " "))

	cmd := exec.Command(args[0], args[1:]...) // #nosec G204
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
