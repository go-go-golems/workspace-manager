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

// RunWSM executes the CLI; uses built binary if present, else go run.
func (s *Sandbox) RunWSM(t *testing.T, ctx context.Context, workDir string, args ...string) RunResult {
    t.Helper()
    moduleRoot := s.projectRoot(t)
    // Prefer prebuilt binary under module root .out from the Dagger build step
    bin := filepath.Join(moduleRoot, ".out", "wsm")
    var cmd *exec.Cmd
    if ctx == nil {
        // Don't use CommandContext when ctx is nil (it panics)
        if _, err := os.Stat(bin); err == nil {
            cmd = exec.Command(bin, args...)
        } else {
            cmd = exec.Command("go", append([]string{"run", "./cmd/wsm"}, args...)...)
            cmd.Dir = moduleRoot
        }
    } else {
        if _, err := os.Stat(bin); err == nil {
            cmd = exec.CommandContext(ctx, bin, args...)
        } else {
            // fall back to go run
            cmd = exec.CommandContext(ctx, "go", append([]string{"run", "./cmd/wsm"}, args...)...)
            cmd.Dir = moduleRoot
        }
    }
    if workDir != "" {
        cmd.Dir = workDir
    }
    cmd.Env = s.Env(nil)

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
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
    return RunResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}
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
            if parent == dir { break }
            dir = parent
        }
    }
    // Fallback to environment or current process CWD
    if wd, err := os.Getwd(); err == nil { return wd }
    return "."
}


