package wsm

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGitInDirOrFail2(t *testing.T, dir string, args ...string) string {
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

// forkedStatusFixture reproduces the WSM-MO-013 failing scenario: a repo
// checked out on a task branch whose configured base branch exists ONLY
// locally (never pushed), so origin/<base> does not exist. Before the fix,
// CheckBranchMerged/CheckBranchNeedsRebase ran git against origin/<base> and
// failed with exit 128, swallowing the error into a confident false. After the
// fix they fall back to the local <base> and return BaseResolved.
func forkedStatusFixture(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "origin.git")
	seed := filepath.Join(tmp, "seed")
	client := filepath.Join(tmp, "client")

	runGitInDirOrFail2(t, "", "init", "--bare", remote)
	runGitInDirOrFail2(t, "", "clone", remote, seed)
	runGitInDirOrFail2(t, seed, "config", "user.name", "WSM Test")
	runGitInDirOrFail2(t, seed, "config", "user.email", "wsm-test@example.com")
	runGitInDirOrFail2(t, seed, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitInDirOrFail2(t, seed, "add", "README.md")
	runGitInDirOrFail2(t, seed, "commit", "-m", "seed commit")
	runGitInDirOrFail2(t, seed, "push", "-u", "origin", "main")

	// Create a local-only base branch (never pushed) and add a commit on it.
	runGitInDirOrFail2(t, seed, "checkout", "-b", "task/deploy-dev-indexer")
	if err := os.WriteFile(filepath.Join(seed, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write base.txt: %v", err)
	}
	runGitInDirOrFail2(t, seed, "add", "base.txt")
	runGitInDirOrFail2(t, seed, "commit", "-m", "local base commit")

	// Clone only main; the task base branch is NOT fetched.
	runGitInDirOrFail2(t, "", "clone", "--branch", "main", remote, client)
	// Recreate the local-only base branch in the client from the client's own main.
	runGitInDirOrFail2(t, client, "branch", "task/deploy-dev-indexer", "main")
	// Add a commit to the base so HEAD..base can be non-empty.
	if err := os.WriteFile(filepath.Join(client, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write client base.txt: %v", err)
	}
	runGitInDirOrFail2(t, client, "checkout", "task/deploy-dev-indexer")
	runGitInDirOrFail2(t, client, "add", "base.txt")
	runGitInDirOrFail2(t, client, "commit", "-m", "client base commit")

	// Now switch to a feature branch cut from the base, to simulate the fork.
	runGitInDirOrFail2(t, client, "checkout", "-b", "task/ragkit-fork", "task/deploy-dev-indexer")
	if err := os.WriteFile(filepath.Join(client, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("write feature.txt: %v", err)
	}
	runGitInDirOrFail2(t, client, "add", "feature.txt")
	runGitInDirOrFail2(t, client, "commit", "-m", "feature commit")

	return client
}

// TestCheckBranchMerged_ForkedWorkspace_LocalFallback is the regression test for
// WSM-MO-013: the configured base branch (task/deploy-dev-indexer) has no
// origin/ ref, but exists locally. The check must resolve via the local
// fallback and return BaseResolved (a real comparison), NOT swallow an exit 128
// into a confident false.
func TestCheckBranchMerged_ForkedWorkspace_LocalFallback(t *testing.T) {
	clientPath := forkedStatusFixture(t)
	c := gitclient.NewCli()
	ctx := context.Background()

	cmp, err := CheckBranchMerged(ctx, c, clientPath, "task/deploy-dev-indexer", "origin")
	require.NoError(t, err)
	assert.Equal(t, BaseResolved, cmp.Status, "should resolve via local fallback, not swallow an error")
	assert.Equal(t, "task/deploy-dev-indexer", cmp.ResolvedRef)
	assert.Equal(t, RefSourceLocal, cmp.RefSource)
	// HEAD (feature commit) is NOT an ancestor of the local base, so not merged.
	assert.False(t, cmp.IsMerged, "feature branch is ahead of base, not merged into it")
	assert.Empty(t, cmp.Reason)
}

// TestCheckBranchNeedsRebase_ForkedWorkspace_LocalFallback mirrors the above for
// the rebase check: it must resolve the local base and report a real
// behind-count instead of returning an error/swallowed false.
func TestCheckBranchNeedsRebase_ForkedWorkspace_LocalFallback(t *testing.T) {
	clientPath := forkedStatusFixture(t)
	c := gitclient.NewCli()
	ctx := context.Background()

	cmp, err := CheckBranchNeedsRebase(ctx, c, clientPath, "task/deploy-dev-indexer", "origin")
	require.NoError(t, err)
	assert.Equal(t, BaseResolved, cmp.Status)
	assert.Equal(t, "task/deploy-dev-indexer", cmp.ResolvedRef)
	assert.Equal(t, RefSourceLocal, cmp.RefSource)
	// HEAD is ahead of the base (feature commit), base is not ahead of HEAD,
	// so needsRebase should be false with a real comparison.
	assert.False(t, cmp.NeedsRebase)
}

// TestCheckBranchMerged_UnknownWhenBaseAbsent asserts the honesty fix: a base
// that exists neither remotely nor locally yields BaseUnknown (NOT a confident
// false), with a reason naming the missing refs.
func TestCheckBranchMerged_UnknownWhenBaseAbsent(t *testing.T) {
	clientPath := forkedStatusFixture(t)
	c := gitclient.NewCli()
	ctx := context.Background()

	cmp, err := CheckBranchMerged(ctx, c, clientPath, "does/not-exist", "origin")
	require.NoError(t, err)
	assert.Equal(t, BaseUnknown, cmp.Status)
	assert.Empty(t, cmp.ResolvedRef)
	assert.False(t, cmp.IsMerged)
	assert.Contains(t, cmp.Reason, "does/not-exist")
	assert.Contains(t, cmp.Reason, "origin")
}

// TestCheckBranchMerged_RemoteTrackingResolved verifies the common
// (non-forked) case still compares against origin/<base>.
func TestCheckBranchMerged_RemoteTrackingResolved(t *testing.T) {
	clientPath := forkedStatusFixture(t)
	c := gitclient.NewCli()
	ctx := context.Background()

	cmp, err := CheckBranchMerged(ctx, c, clientPath, "main", "origin")
	require.NoError(t, err)
	assert.Equal(t, BaseResolved, cmp.Status)
	assert.Equal(t, "origin/main", cmp.ResolvedRef)
	assert.Equal(t, RefSourceRemoteTracking, cmp.RefSource)
}

// TestCheckBranchNeedsRebase_SkipWhenOnBase asserts the existing skip-on-base
// behavior is preserved: when HEAD is the base branch itself, no rebase is
// needed and NeedsRebase is false.
func TestCheckBranchNeedsRebase_SkipWhenOnBase(t *testing.T) {
	clientPath := forkedStatusFixture(t)
	c := gitclient.NewCli()
	ctx := context.Background()

	// Check out the base branch itself.
	runGitInDirOrFail2(t, clientPath, "checkout", "task/deploy-dev-indexer")

	cmp, err := CheckBranchNeedsRebase(ctx, c, clientPath, "task/deploy-dev-indexer", "origin")
	require.NoError(t, err)
	assert.Equal(t, BaseResolved, cmp.Status)
	assert.False(t, cmp.NeedsRebase, "already on the base branch -> no rebase needed")
}
