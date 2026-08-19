package wsm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOverlayWorkspaceBaseOverrides_MergesInWorkspaceOverConfigDir verifies the
// E3 local-beats-global overlay: per-repo BaseBranch/BaseRemote in the
// in-workspace .wsm/wsm.json become BaseBranchWorkspace/BaseRemoteWorkspace on
// the loaded Repository, while repos without an override keep their config-dir
// (global) values untouched.
func TestOverlayWorkspaceBaseOverrides_MergesInWorkspaceOverConfigDir(t *testing.T) {
	tmp := t.TempDir()
	wsPath := filepath.Join(tmp, "ws")
	require.NoError(t, os.MkdirAll(filepath.Join(wsPath, ".wsm"), 0o755))

	meta := WorkspaceMetadata{
		Name: "ws",
		Path: wsPath,
		Repositories: []RepositoryMetadata{
			{Name: "goldeneaglecoin.com", BaseBranch: "develop", BaseRemote: "origin"},
			{Name: "geppetto", BaseBranch: "main", BaseRemote: "upstream"},
			// coinvault: no override -> should stay empty (inherit).
			{Name: "coinvault"},
		},
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(wsPath, ".wsm", "wsm.json"), data, 0o644))

	ws := &Workspace{
		Name: "ws",
		Path: wsPath,
		Repositories: []Repository{
			{Name: "goldeneaglecoin.com", BaseBranch: "main"}, // global override present
			{Name: "geppetto"}, // no global override
			{Name: "coinvault", DefaultBaseBranch: "main"}, // no override anywhere
		},
	}

	overlayWorkspaceBaseOverrides(ws)

	assert.Equal(t, "develop", ws.Repositories[0].BaseBranchWorkspace)
	assert.Equal(t, "origin", ws.Repositories[0].BaseRemoteWorkspace) // explicitly set in wsm.json
	assert.Equal(t, "main", ws.Repositories[0].BaseBranch, "global value preserved separately")
	assert.Equal(t, "main", ws.Repositories[1].BaseBranchWorkspace)
	assert.Equal(t, "upstream", ws.Repositories[1].BaseRemoteWorkspace)
	assert.Empty(t, ws.Repositories[2].BaseBranchWorkspace, "no override -> stays empty (inherits)")
	assert.Equal(t, "main", ws.Repositories[2].DefaultBaseBranch, "discovered default untouched")
}

func TestOverlayWorkspaceBaseOverrides_MissingFileIsNonFatal(t *testing.T) {
	ws := &Workspace{
		Name:         "ws",
		Path:         t.TempDir(), // no .wsm/wsm.json here
		Repositories: []Repository{{Name: "repo", BaseBranch: "main"}},
	}
	// Must not panic or error.
	overlayWorkspaceBaseOverrides(ws)
	assert.Empty(t, ws.Repositories[0].BaseBranchWorkspace)
	assert.Equal(t, "main", ws.Repositories[0].BaseBranch, "config-dir value untouched when no in-workspace file")
}

func TestOverlayWorkspaceBaseOverrides_NilSafe(t *testing.T) {
	assert.NotPanics(t, func() { overlayWorkspaceBaseOverrides(nil) })
	assert.NotPanics(t, func() { overlayWorkspaceBaseOverrides(&Workspace{}) })
}

// TestFillMissingDefaultBaseBranches_DetectsFromRepoPath verifies the P1
// self-heal: a workspace whose Repository.DefaultBaseBranch is empty (created
// before discovery) gets it filled from the repo's actual remote default on
// load, so status does not fall through to "main" for a develop-default repo.
func TestFillMissingDefaultBaseBranches_DetectsFromRepoPath(t *testing.T) {
	tmp := t.TempDir()
	// Build a repo whose remote advertises "develop" as its default.
	remote := filepath.Join(tmp, "origin.git")
	seed := filepath.Join(tmp, "seed")
	repo := filepath.Join(tmp, "repo")
	runGitInDirOrFail2(t, "", "init", "--bare", remote)
	runGitInDirOrFail2(t, remote, "symbolic-ref", "HEAD", "refs/heads/develop")
	runGitInDirOrFail2(t, "", "clone", remote, seed)
	runGitInDirOrFail2(t, seed, "config", "user.name", "WSM Test")
	runGitInDirOrFail2(t, seed, "config", "user.email", "wsm-test@example.com")
	runGitInDirOrFail2(t, seed, "checkout", "-b", "develop")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitInDirOrFail2(t, seed, "add", "README.md")
	runGitInDirOrFail2(t, seed, "commit", "-m", "seed commit")
	runGitInDirOrFail2(t, seed, "push", "-u", "origin", "develop")
	runGitInDirOrFail2(t, "", "clone", remote, repo)

	ws := &Workspace{
		Name: "ws",
		Path: tmp,
		Repositories: []Repository{
			{Name: "repo", Path: repo, DefaultBaseBranch: ""}, // stale/empty, as pre-discovery
		},
	}
	fillMissingDefaultBaseBranches(context.Background(), ws)
	assert.Equal(t, "develop", ws.Repositories[0].DefaultBaseBranch,
		"empty DefaultBaseBranch should be detected from the repo's remote default on load")
}

// TestFillMissingDefaultBaseBranches_PreservesExisting ensures a repo that
// already has a DefaultBaseBranch is not re-detected (avoids surprise git
// calls and preserves an explicit value).
func TestFillMissingDefaultBaseBranches_PreservesExisting(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	runGitInDirOrFail2(t, "", "init", repo)
	ws := &Workspace{
		Name: "ws",
		Path: tmp,
		Repositories: []Repository{
			{Name: "repo", Path: repo, DefaultBaseBranch: "main"},
		},
	}
	fillMissingDefaultBaseBranches(context.Background(), ws)
	assert.Equal(t, "main", ws.Repositories[0].DefaultBaseBranch, "existing value preserved")
}

// TestFillMissingDefaultBaseBranches_MissingRepoPathIsSkipped ensures a repo
// whose path does not exist is left untouched (no panic, no fill).
func TestFillMissingDefaultBaseBranches_MissingRepoPathIsSkipped(t *testing.T) {
	ws := &Workspace{
		Name: "ws",
		Path: t.TempDir(),
		Repositories: []Repository{
			{Name: "gone", Path: filepath.Join(t.TempDir(), "does-not-exist"), DefaultBaseBranch: ""},
		},
	}
	fillMissingDefaultBaseBranches(context.Background(), ws)
	assert.Empty(t, ws.Repositories[0].DefaultBaseBranch, "missing repo path -> left empty, no error")
}
