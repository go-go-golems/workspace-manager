package gitclient

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGitOrFail(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func createRemoteBranchFixture(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "origin.git")
	seed := filepath.Join(tmp, "seed")
	client := filepath.Join(tmp, "client")

	runGitOrFail(t, "", "init", "--bare", remote)
	runGitOrFail(t, "", "clone", remote, seed)
	runGitOrFail(t, seed, "config", "user.name", "WSM Test")
	runGitOrFail(t, seed, "config", "user.email", "wsm-test@example.com")
	runGitOrFail(t, seed, "checkout", "-b", "main")

	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitOrFail(t, seed, "add", "README.md")
	runGitOrFail(t, seed, "commit", "-m", "seed commit")
	runGitOrFail(t, seed, "push", "-u", "origin", "main")

	runGitOrFail(t, seed, "checkout", "-b", "feature/remote-only")
	if err := os.WriteFile(filepath.Join(seed, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("write feature.txt: %v", err)
	}
	runGitOrFail(t, seed, "add", "feature.txt")
	runGitOrFail(t, seed, "commit", "-m", "remote branch commit")
	runGitOrFail(t, seed, "push", "-u", "origin", "feature/remote-only")

	runGitOrFail(t, "", "clone", "--branch", "main", remote, client)
	return client
}

func TestCliRemoteTrackingBranchExists(t *testing.T) {
	clientPath := createRemoteBranchFixture(t)
	c := NewCli()
	ctx := context.Background()

	h, err := c.Open(ctx, clientPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ok, err := c.RemoteTrackingBranchExists(ctx, h, "origin", "feature/remote-only")
	if err != nil {
		t.Fatalf("remote branch exists failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected remote branch to exist")
	}

	ok, err = c.RemoteTrackingBranchExists(ctx, h, "origin", "feature/missing")
	if err != nil {
		t.Fatalf("remote branch exists (missing) failed: %v", err)
	}
	if ok {
		t.Fatalf("expected missing branch to return false")
	}
}

func TestGoGitRemoteTrackingBranchExists(t *testing.T) {
	clientPath := createRemoteBranchFixture(t)
	c := NewGoGit()
	ctx := context.Background()

	h, err := c.Open(ctx, clientPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ok, err := c.RemoteTrackingBranchExists(ctx, h, "origin", "feature/remote-only")
	if err != nil {
		t.Fatalf("remote branch exists failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected remote branch to exist")
	}

	ok, err = c.RemoteTrackingBranchExists(ctx, h, "origin", "feature/missing")
	if err != nil {
		t.Fatalf("remote branch exists (missing) failed: %v", err)
	}
	if ok {
		t.Fatalf("expected missing branch to return false")
	}
}
