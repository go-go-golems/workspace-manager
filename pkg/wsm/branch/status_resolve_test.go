package branch

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

// baseRefFixture builds a clone of a bare remote with "main" pushed, plus a
// local-only task branch (never pushed) that exists in the client worktree only.
// It returns the client repo path. This shape reproduces the forked-workspace
// scenario where the configured base branch has no remote-tracking ref.
func baseRefFixture(t *testing.T) string {
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

	// Local-only base branch: created in the seed, NOT pushed. The client clones
	// only main, so the client has origin/main but no origin/task/deploy-dev-indexer.
	runGitOrFail(t, seed, "checkout", "-b", "task/deploy-dev-indexer")
	if err := os.WriteFile(filepath.Join(seed, "task.txt"), []byte("task\n"), 0o644); err != nil {
		t.Fatalf("write task.txt: %v", err)
	}
	runGitOrFail(t, seed, "add", "task.txt")
	runGitOrFail(t, seed, "commit", "-m", "local task branch commit")

	// Clone only main; the local task branch is NOT fetched (it was never pushed),
	// so the client has origin/main but no origin/task/deploy-dev-indexer.
	runGitOrFail(t, "", "clone", "--branch", "main", remote, client)

	// Create the local-only base branch in the client from the client's own main
	// tip, so refs/heads/task/deploy-dev-indexer exists but refs/remotes/origin/...
	// does not (it was never pushed).
	runGitOrFail(t, client, "branch", "task/deploy-dev-indexer", "main")

	return client
}

func TestResolveBaseRef_RemoteTrackingPreferred(t *testing.T) {
	clientPath := baseRefFixture(t)
	c := gitclient.NewCli()
	ctx := context.Background()

	// "main" exists as origin/main (remote-tracking) -> preferred over local.
	res, err := ResolveBaseRef(ctx, c, clientPath, "main", "origin")
	require.NoError(t, err)
	assert.Equal(t, BaseResolved, res.Status)
	assert.Equal(t, "origin/main", res.Ref)
	assert.Equal(t, RefSourceRemoteTracking, res.Source)
	assert.Empty(t, res.Reason)
}

func TestResolveBaseRef_LocalFallback(t *testing.T) {
	clientPath := baseRefFixture(t)
	c := gitclient.NewCli()
	ctx := context.Background()

	// task/deploy-dev-indexer has NO origin/ ref but DOES exist locally.
	// This is the forked-workspace case: resolve to the local branch.
	res, err := ResolveBaseRef(ctx, c, clientPath, "task/deploy-dev-indexer", "origin")
	require.NoError(t, err)
	assert.Equal(t, BaseResolved, res.Status)
	assert.Equal(t, "task/deploy-dev-indexer", res.Ref)
	assert.Equal(t, RefSourceLocal, res.Source)
	assert.Empty(t, res.Reason)
}

func TestResolveBaseRef_UnknownWhenAbsent(t *testing.T) {
	clientPath := baseRefFixture(t)
	c := gitclient.NewCli()
	ctx := context.Background()

	// A branch that exists neither remotely nor locally.
	res, err := ResolveBaseRef(ctx, c, clientPath, "does/not-exist", "origin")
	require.NoError(t, err)
	assert.Equal(t, BaseUnknown, res.Status)
	assert.Empty(t, res.Ref)
	assert.Empty(t, res.Source)
	assert.Contains(t, res.Reason, "does/not-exist")
	assert.Contains(t, res.Reason, "origin")
	assert.Contains(t, res.Reason, "not a local branch")
}

func TestResolveBaseRef_EmptyBaseIsUnknown(t *testing.T) {
	clientPath := baseRefFixture(t)
	c := gitclient.NewCli()
	ctx := context.Background()

	res, err := ResolveBaseRef(ctx, c, clientPath, "", "origin")
	require.NoError(t, err)
	assert.Equal(t, BaseUnknown, res.Status)
	assert.Contains(t, res.Reason, "empty")
}

func TestResolveBaseRef_DefaultRemoteWhenEmpty(t *testing.T) {
	clientPath := baseRefFixture(t)
	c := gitclient.NewCli()
	ctx := context.Background()

	// Passing empty remote should normalize to "origin" and still resolve origin/main.
	res, err := ResolveBaseRef(ctx, c, clientPath, "main", "")
	require.NoError(t, err)
	assert.Equal(t, BaseResolved, res.Status)
	assert.Equal(t, "origin/main", res.Ref)
	assert.Equal(t, RefSourceRemoteTracking, res.Source)
}

func TestDistinctBranches(t *testing.T) {
	branches := map[string]string{
		"coinvault": "task/deploy-dev-indexer",
		"geppetto":  "task/deploy-dev-indexer",
		"goldeneag": "task/deploy-image",
	}
	assert.Equal(t, []string{"task/deploy-dev-indexer", "task/deploy-image"}, DistinctBranches(branches))
}

func TestMostFrequentBranch(t *testing.T) {
	branches := map[string]string{
		"coinvault": "task/deploy-dev-indexer",
		"geppetto":  "task/deploy-dev-indexer",
		"goldeneag": "task/deploy-image",
	}
	assert.Equal(t, "task/deploy-dev-indexer", MostFrequentBranch(branches))
}

func TestMostFrequentBranch_TieBreaksBySortedOrder(t *testing.T) {
	branches := map[string]string{
		"a": "beta",
		"b": "alpha",
	}
	// Equal counts -> sorted-first wins ("alpha" < "beta").
	assert.Equal(t, "alpha", MostFrequentBranch(branches))
}

func TestMostFrequentBranch_Empty(t *testing.T) {
	assert.Equal(t, "", MostFrequentBranch(map[string]string{}))
}
