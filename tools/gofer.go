//go:build tools

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func main() {
	flags := flag.NewFlagSet("gofer <command>", flag.ExitOnError)
	err := flags.Parse(os.Args[1:])
	if err != nil {
		log.Fatalf("%s", err)
	}

	cmd := map[string]func([]string) error{
		"sandbox":  sandbox,
		"mod-path": modPath,
	}

	args := flags.Args()
	if len(args) < 1 {
		flags.Usage()
		log.Fatalf("missing command")
	}

	fn, ok := cmd[args[0]]
	if !ok {
		log.Fatalf("invalid command %s", args[0])
	}

	err = fn(args[1:])
	if err != nil {
		log.Fatalf("%s", err)
	}
}

func sandbox(args []string) error {
	flags := flag.NewFlagSet("gofer sandbox <exec>", flag.ExitOnError)

	// These flags can't conveniently be set by the calling process because
	// e.g., `GOOS=windows go run gofer.go sandbox` on a non-Windows
	// machine breaks the execution of gofer.go.
	goarch := flags.String("arch", "", "GOARCH for the sandboxed process")
	goos := flags.String("os", "", "GOOS for the sandboxed process")

	err := flags.Parse(args)
	if err != nil {
		return err
	}

	remainingArgs := flags.Args()
	if len(remainingArgs) < 1 {
		flags.Usage()
		return fmt.Errorf("missing command")
	}

	docker, err := hasPath("/.dockerenv")
	if err != nil {
		return err
	}

	podman, err := hasPath("/run/.containerenv")
	if err != nil {
		return err
	}

	runCmd := []string{}
	if runtime.GOOS != "linux" || docker || podman {
		log.Println("WARNING: running non-sandboxed")
	} else {
		runCmd = append(runCmd, []string{
			"bwrap",
			"--new-session",
			"--die-with-parent",
			"--unshare-user",
			"--unshare-ipc",
			"--unshare-pid",
			"--unshare-net",
			"--unshare-uts",
			"--unshare-cgroup",
			"--proc", "/proc",
			"--dev", "/dev",
			"--tmpfs", "/tmp",
			"--ro-bind-try", "/usr", "/usr",
			"--ro-bind-try", "/lib", "/lib",
			"--ro-bind-try", "/lib32", "/lib32",
			"--ro-bind-try", "/lib64", "/lib64",
		}...)

		exists, err := hasPath("replace")
		if err != nil {
			return err
		}
		if exists {
			entries, err := os.ReadDir("replace")
			if err != nil {
				return err
			}

			for _, entry := range entries {
				path, err := filepath.Abs(filepath.Join("replace", entry.Name()))
				if err != nil {
					return err
				}

				info, err := entry.Info()
				if err != nil {
					return err
				}

				if info.Mode()&os.ModeSymlink == os.ModeSymlink {
					dst, err := os.Readlink(path)
					if err != nil {
						return err
					}

					path, err = filepath.Abs(filepath.Join("replace", dst))
					if err != nil {
						return err
					}
				}

				runCmd = append(runCmd, "--ro-bind-try", path, path)
			}
		}

		project, err := projectPath()
		if err != nil {
			return err
		}
		runCmd = append(runCmd, "--bind", project, project)

		replacers, err := replacePaths(project)
		if err != nil {
			return err
		}
		for _, replace := range replacers {
			runCmd = append(runCmd, "--ro-bind-try", replace, replace)
		}

		goPath, err := goEnv("GOPATH")
		if err != nil {
			return err
		}
		runCmd = append(runCmd, "--bind-try", goPath, goPath)

		goCache, err := goEnv("GOCACHE")
		if err != nil {
			return err
		}
		runCmd = append(runCmd, "--bind-try", goCache, goCache)

		output := os.Getenv("OUTPUT")
		if output != "" {
			runCmd = append(runCmd, "--bind-try", output, output)
		}

		outputTools := os.Getenv("OUTPUT_TOOLS")
		if outputTools != "" {
			runCmd = append(runCmd, "--bind-try", outputTools, outputTools)
		}
	}
	runCmd = append(runCmd, remainingArgs...)

	run := exec.Command(runCmd[0], runCmd[1:]...) // #nosec G204
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	run.Env = os.Environ()
	if *goarch != "" {
		run.Env = append(run.Env, fmt.Sprintf("GOARCH=%s", *goarch))
	}
	if *goos != "" {
		run.Env = append(run.Env, fmt.Sprintf("GOOS=%s", *goos))
	}

	err = run.Run()
	if err != nil {
		return err
	}
	return nil
}

func modPath(args []string) error {
	flags := flag.NewFlagSet("gofer mod-path -mod MODULE-NAME", flag.ExitOnError)
	mod := flags.String("mod", "", "module name")

	err := flags.Parse(args)
	if err != nil {
		return err
	}

	if *mod == "" {
		flags.Usage()
		return fmt.Errorf("missing module name")
	}

	project, err := projectPath()
	if err != nil {
		return err
	}

	sums, err := os.ReadFile(filepath.Join(project, "go.sum"))
	if err != nil {
		return err
	}

	version := ""
	for _, line := range splitLines(string(sums)) {
		parts := strings.Split(line, " ")
		if len(parts) != 3 {
			return fmt.Errorf("invalid line: %s", line)
		}

		if parts[0] == *mod && !strings.HasSuffix(parts[1], "/go.mod") {
			version = parts[1]
		}
	}

	if version == "" {
		return fmt.Errorf("missing entry for %s", *mod)
	}

	goPath, err := goEnv("GOPATH")
	if err != nil {
		return err
	}

	modPath := filepath.Join(strings.Split(*mod, "/")...)
	fmt.Println(filepath.Join(goPath, "pkg", "mod", fmt.Sprintf("%s@%s", modPath, version)))
	return nil
}

func hasPath(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func projectPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime error")
	}

	dir, err := filepath.Abs(filepath.Join(file, "..", ".."))
	if err != nil {
		return "", err
	}

	return dir, nil
}

func replacePaths(project string) ([]string, error) {
	rx, err := regexp.Compile("^replace [^ ]+( v[^ ]+)? => ([^ ]+)( v[^ ]+)?$")
	if err != nil {
		return nil, err
	}

	mod, err := os.ReadFile(filepath.Join(project, "go.mod")) // #nosec G304
	if err != nil {
		return nil, err
	}

	paths := []string{}
	for _, line := range strings.Split(string(mod), "\n") {
		matches := rx.FindStringSubmatch(line)
		if len(matches) == 4 {
			path, err := filepath.Abs(filepath.Join(project, matches[2]))
			if err != nil {
				return nil, err
			}
			paths = append(paths, path)
		}
	}

	return paths, nil
}

func goEnv(key string) (string, error) {
	cmd := exec.Command("go", "env", key)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	value := strings.Trim(string(output), "\r\n")
	if value == "" {
		return "", fmt.Errorf("%s is not set", key)
	}

	return value, nil
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimRight(s, "\n")
	return strings.Split(s, "\n")
}
