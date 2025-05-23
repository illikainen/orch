//go:generate go run ./tools/generate.go

package main

import (
	"os"

	"github.com/illikainen/orch/src/cmd"

	"github.com/fatih/color"
	"github.com/illikainen/go-utils/src/logging"
	"github.com/illikainen/go-utils/src/sandbox"
	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
)

func main() {
	color.NoColor = !isatty.IsTerminal(os.Stderr.Fd())
	log.SetOutput(os.Stderr)

	if !sandbox.IsSandboxed() {
		log.SetFormatter(&logging.SanitizedTextFormatter{})
	}

	err := cmd.Command().Execute()
	if err != nil {
		log.Tracef("%+v", err)
		log.Fatalf("%s", err)
	}
}
