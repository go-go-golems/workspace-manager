package helpers

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type RunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// BuildWSM builds the CLI into .out/wsm and returns the path to the binary.
func (s *Sandbox) BuildWSM(t *testing.T, ctx context.Context) string {
	t.Helper()
	outDir := filepath.Join(s.BaseDir, ".out")
	run(t, s, s.BaseDir, nil, "mkdir", "-p", outDir)
	run(t, s, s.projectRoot(t), nil, "go", "build", "-o", filepath.Join(outDir, "wsm"), "./cmd/wsm")
	return filepath.Join(outDir, "wsm")
}

// RunWSM executes the CLI.
// By default it builds/uses a sandbox-local binary to ensure tests exercise current source.
// Set WSM_TEST_USE_PREBUILT=1 to opt into moduleRoot/.out/wsm when available.
func (s *Sandbox) RunWSM(t *testing.T, ctx context.Context, workDir string, args ...string) RunResult {
	t.Helper()
	moduleRoot := s.projectRoot(t)
	useExternalPrebuilt := os.Getenv("WSM_TEST_USE_PREBUILT") == "1"
	externalBin := filepath.Join(moduleRoot, ".out", "wsm")
	localBin := filepath.Join(s.BaseDir, ".out", "wsm")

	// Default path: build a fresh binary once per sandbox and execute that.
	bin := localBin
	if useExternalPrebuilt {
		if _, err := os.Stat(externalBin); err == nil {
			bin = externalBin
		}
	}
	if _, err := os.Stat(bin); os.IsNotExist(err) {
		bin = s.BuildWSM(t, ctx)
	}

	var cmd *exec.Cmd
	// #nosec G204 -- test helper running the built wsm binary with test-controlled args; not production code.
	if ctx == nil {
		cmd = exec.Command(bin, args...)
	} else {
		cmd = exec.CommandContext(ctx, bin, args...)
	}
	cmd.Dir = moduleRoot
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = s.Env(nil)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Log command and CWD
	t.Logf("[cmd] cwd=%s args=%q", cmd.Dir, strings.Join(cmd.Args, " "))

	err := cmd.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			// Treat other errors as exit code 1
			code = 1
		}
	}

	res := RunResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
	if res.ExitCode != 0 {
		// Emit tails for easier debugging
		tail := func(s string) string {
			lines := strings.Split(s, "\n")
			if len(lines) > 20 {
				lines = lines[len(lines)-20:]
			}
			return strings.Join(lines, "\n")
		}
		t.Logf("[cmd] exit=%d\nstdout(last 20):\n%s\nstderr(last 20):\n%s", res.ExitCode, tail(res.Stdout), tail(res.Stderr))
	}
	return res
}

// projectRoot returns the repo root (assumes tests live in workspace-manager/test/integration/...)
func (s *Sandbox) projectRoot(t *testing.T) string {
	t.Helper()
	// Try to locate the module root by walking up until we find go.mod with our module path
	wd, err := os.Getwd()
	if err == nil {
		dir := wd
		for i := 0; i < 10; i++ {
			gomod := filepath.Join(dir, "go.mod")
			if data, err := os.ReadFile(gomod); err == nil {
				if strings.Contains(string(data), "module github.com/go-go-golems/workspace-manager") {
					return dir
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	// Fallback to environment or current process CWD
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}
