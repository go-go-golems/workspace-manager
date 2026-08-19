package gitclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultBranchFixture builds a bare remote advertising a "develop" default
// branch (via symbolic-ref HEAD) plus a client clone, and a second remote
// whose HEAD is unset, to exercise both the symbolic-ref path and the
// unset-HEAD fallback.
func defaultBranchFixture(t *testing.T) (developClient string, unsetClient string) {
	t.Helper()
	tmp := t.TempDir()

	// Remote A: default branch = develop (advertised via HEAD).
	remoteA := filepath.Join(tmp, "originA.git")
	seedA := filepath.Join(tmp, "seedA")
	developClient = filepath.Join(tmp, "clientA")
	runGitOrFail(t, "", "init", "--bare", remoteA)
	// Point the bare remote's HEAD at develop before any push.
	runGitOrFail(t, remoteA, "symbolic-ref", "HEAD", "refs/heads/develop")
	runGitOrFail(t, "", "clone", remoteA, seedA)
	runGitOrFail(t, seedA, "config", "user.name", "WSM Test")
	runGitOrFail(t, seedA, "config", "user.email", "wsm-test@example.com")
	runGitOrFail(t, seedA, "checkout", "-b", "develop")
	if err := os.WriteFile(filepath.Join(seedA, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitOrFail(t, seedA, "add", "README.md")
	runGitOrFail(t, seedA, "commit", "-m", "seed commit")
	runGitOrFail(t, seedA, "push", "-u", "origin", "develop")
	// Clone normally so origin/HEAD is set to origin/develop.
	runGitOrFail(t, "", "clone", remoteA, developClient)

	// Remote B: default branch unset (no symbolic-ref HEAD).
	remoteB := filepath.Join(tmp, "originB.git")
	seedB := filepath.Join(tmp, "seedB")
	unsetClient = filepath.Join(tmp, "clientB")
	runGitOrFail(t, "", "init", "--bare", remoteB)
	runGitOrFail(t, "", "clone", remoteB, seedB)
	runGitOrFail(t, seedB, "config", "user.name", "WSM Test")
	runGitOrFail(t, seedB, "config", "user.email", "wsm-test@example.com")
	runGitOrFail(t, seedB, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(seedB, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitOrFail(t, seedB, "add", "README.md")
	runGitOrFail(t, seedB, "commit", "-m", "seed commit")
	runGitOrFail(t, seedB, "push", "-u", "origin", "main")
	runGitOrFail(t, "", "clone", "--branch", "main", remoteB, unsetClient)
	// Remove origin/HEAD so the client has no advertised default (simulates a
	// remote that never set HEAD and a clone that didn't synthesize it).
	runGitOrFail(t, unsetClient, "symbolic-ref", "-d", "refs/remotes/origin/HEAD")

	return developClient, unsetClient
}

func TestCliDefaultBranch_Advertised(t *testing.T) {
	developClient, _ := defaultBranchFixture(t)
	c := NewCli()
	ctx := context.Background()

	h, err := c.Open(ctx, developClient)
	require.NoError(t, err)

	got, err := c.DefaultBranch(ctx, h, "origin")
	require.NoError(t, err)
	assert.Equal(t, "develop", got, "remote advertised develop as its default via HEAD")
}

func TestCliDefaultBranch_UnsetReturnsEmpty(t *testing.T) {
	_, unsetClient := defaultBranchFixture(t)
	c := NewCli()
	ctx := context.Background()

	h, err := c.Open(ctx, unsetClient)
	require.NoError(t, err)

	got, err := c.DefaultBranch(ctx, h, "origin")
	require.NoError(t, err, "unset HEAD is not an error")
	assert.Empty(t, got, "unset origin/HEAD should yield empty string, not an error")
}

func TestCliDefaultBranch_DefaultRemoteWhenEmpty(t *testing.T) {
	developClient, _ := defaultBranchFixture(t)
	c := NewCli()
	ctx := context.Background()

	h, err := c.Open(ctx, developClient)
	require.NoError(t, err)

	got, err := c.DefaultBranch(ctx, h, "")
	require.NoError(t, err)
	assert.Equal(t, "develop", got, "empty remote should normalize to origin")
}
