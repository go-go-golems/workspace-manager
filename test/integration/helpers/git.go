package helpers

import (
    "bytes"
    "fmt"
    "os/exec"
    "path/filepath"
    "testing"
)

// InitBareRepo initializes a bare remote repository under s.RemotesDir and returns its file:// URL.
func (s *Sandbox) InitBareRepo(t *testing.T, name string) string {
    t.Helper()
    path := filepath.Join(s.RemotesDir, name+".git")
    run(t, s, s.BaseDir, nil, "git", "init", "--bare", path)
    return "file://" + path
}

// InitRepo initializes a local repo and optionally sets origin to remoteURL. Returns path.
func (s *Sandbox) InitRepo(t *testing.T, name string, remoteURL string) string {
    t.Helper()
    path := filepath.Join(s.ReposDir, name)
    run(t, s, s.BaseDir, nil, "mkdir", "-p", path)
    run(t, s, path, nil, "git", "init")
    run(t, s, path, nil, "git", "config", "user.name", "WSM Test")
    run(t, s, path, nil, "git", "config", "user.email", "wsm@example.com")
    if remoteURL != "" {
        run(t, s, path, nil, "git", "remote", "add", "origin", remoteURL)
        // Establish main branch
        run(t, s, path, nil, "git", "checkout", "-b", "main")
        s.CommitFile(t, path, "README.md", "initial", "initial commit")
        run(t, s, path, nil, "git", "push", "-u", "origin", "main")
    }
    return path
}

func (s *Sandbox) CommitFile(t *testing.T, repoPath, filename, content, message string) {
    t.Helper()
    run(t, s, repoPath, nil, "bash", "-lc", fmt.Sprintf("printf %q > %s", content, filename))
    run(t, s, repoPath, nil, "git", "add", filename)
    run(t, s, repoPath, nil, "git", "commit", "-m", message)
}

func (s *Sandbox) CreateBranch(t *testing.T, repoPath, name, base string) {
    t.Helper()
    if base == "" { base = "main" }
    run(t, s, repoPath, nil, "git", "checkout", base)
    run(t, s, repoPath, nil, "git", "checkout", "-b", name)
}

func (s *Sandbox) Checkout(t *testing.T, repoPath, name string) {
    t.Helper()
    run(t, s, repoPath, nil, "git", "checkout", name)
}

func (s *Sandbox) IntroduceConflict(t *testing.T, repoPath, filename string) {
    t.Helper()
    // Create a conflicting change by modifying the same file twice in diverged branches.
    // Caller should set up branches and merges/rebase to trigger conflicts.
    run(t, s, repoPath, nil, "bash", "-lc", fmt.Sprintf("printf %q > %s", "conflict", filename))
    run(t, s, repoPath, nil, "git", "add", filename)
    run(t, s, repoPath, nil, "git", "commit", "-m", "conflict")
}

// RunForTest is an exported wrapper to execute commands with sandbox env in tests.
func RunForTest(t *testing.T, s *Sandbox, dir string, extraEnv map[string]string, name string, args ...string) string {
    return run(t, s, dir, extraEnv, name, args...)
}

func run(t *testing.T, s *Sandbox, dir string, extraEnv map[string]string, name string, args ...string) string {
    t.Helper()
    cmd := exec.Command(name, args...)
    cmd.Dir = dir
    cmd.Env = s.Env(extraEnv)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    if err := cmd.Run(); err != nil {
        t.Fatalf("cmd failed: %s %v\nstdout: %s\nstderr: %s\nerr: %v", name, args, stdout.String(), stderr.String(), err)
    }
    return stdout.String()
}


