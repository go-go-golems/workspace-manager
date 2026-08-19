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

// newSetBaseTestWorkspace builds a minimal on-disk workspace + config-dir JSON
// and returns a WorkspaceManager pointed at it. Used to test SetRepoBase's two
// stores (in-workspace vs config-dir) and the overlay merge end-to-end.
func newSetBaseTestWorkspace(t *testing.T, repoName string) (wm *WorkspaceManager, wsPath string, configPath string) {
	t.Helper()
	tmp := t.TempDir()
	wsPath = filepath.Join(tmp, "ws")
	require.NoError(t, os.MkdirAll(filepath.Join(wsPath, ".wsm"), 0o755))

	// In-workspace metadata with the repo present (no override yet).
	meta := WorkspaceMetadata{
		Name: "ws",
		Path: wsPath,
		Repositories: []RepositoryMetadata{
			{Name: repoName, WorktreePath: filepath.Join(wsPath, repoName)},
		},
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(wsPath, ".wsm", "wsm.json"), data, 0o644))

	wm, err = NewWorkspaceManager()
	require.NoError(t, err)

	// Write the config-dir workspace JSON directly via SaveWorkspace so the
	// manager can load it.
	ws := &Workspace{
		Name:   "ws",
		Path:   wsPath,
		Branch: "task/ws",
	}
	ws.Repositories = []Repository{{Name: repoName, Path: wsPath}}
	require.NoError(t, wm.SaveWorkspace(ws))

	configDir, err := os.UserConfigDir()
	require.NoError(t, err)
	configPath = filepath.Join(configDir, "workspace-manager", "workspaces", "ws.json")
	return wm, wsPath, configPath
}

func TestSetRepoBase_DefaultWritesInWorkspaceOnly(t *testing.T) {
	wm, wsPath, configPath := newSetBaseTestWorkspace(t, "repo1")
	ctx := context.Background()

	require.NoError(t, wm.SetRepoBase(ctx, "ws", SetRepoBaseOptions{
		RepoName: "repo1",
		Branch:   "develop",
	}))

	// In-workspace file has the override.
	metaData, err := os.ReadFile(filepath.Join(wsPath, ".wsm", "wsm.json"))
	require.NoError(t, err)
	var meta WorkspaceMetadata
	require.NoError(t, json.Unmarshal(metaData, &meta))
	assert.Equal(t, "develop", meta.Repositories[0].BaseBranch)

	// Config-dir JSON is NOT touched (no base_branch field set).
	cfgData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var cfgWS Workspace
	require.NoError(t, json.Unmarshal(cfgData, &cfgWS))
	assert.Empty(t, cfgWS.Repositories[0].BaseBranch, "default mode must not mirror to config-dir")
}

func TestSetRepoBase_GlobalWritesConfigDirOnly(t *testing.T) {
	wm, _, configPath := newSetBaseTestWorkspace(t, "repo1")
	ctx := context.Background()

	require.NoError(t, wm.SetRepoBase(ctx, "ws", SetRepoBaseOptions{
		RepoName: "repo1",
		Branch:   "develop",
		Remote:   "upstream",
		Global:   true,
	}))

	// Config-dir JSON has the override.
	cfgData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var cfgWS Workspace
	require.NoError(t, json.Unmarshal(cfgData, &cfgWS))
	assert.Equal(t, "develop", cfgWS.Repositories[0].BaseBranch)
	assert.Equal(t, "upstream", cfgWS.Repositories[0].BaseRemote)

	// In-workspace file is NOT touched (no baseBranch).
	loaded, err := wm.LoadWorkspace("ws")
	require.NoError(t, err)
	metaData, err := os.ReadFile(filepath.Join(loaded.Path, ".wsm", "wsm.json"))
	require.NoError(t, err)
	var meta WorkspaceMetadata
	require.NoError(t, json.Unmarshal(metaData, &meta))
	assert.Empty(t, meta.Repositories[0].BaseBranch, "--global must not touch .wsm/wsm.json")
}

func TestSetRepoBase_LocalBeatsGlobalAfterLoad(t *testing.T) {
	wm, _, _ := newSetBaseTestWorkspace(t, "repo1")
	ctx := context.Background()

	// Set a global override first, then a local one.
	require.NoError(t, wm.SetRepoBase(ctx, "ws", SetRepoBaseOptions{RepoName: "repo1", Branch: "main", Global: true}))
	require.NoError(t, wm.SetRepoBase(ctx, "ws", SetRepoBaseOptions{RepoName: "repo1", Branch: "develop"}))

	loaded, err := wm.LoadWorkspace("ws")
	require.NoError(t, err)
	// The overlay should make the in-workspace "develop" win over config-dir "main".
	assert.Equal(t, "develop", loaded.Repositories[0].BaseBranchWorkspace)
	assert.Equal(t, "main", loaded.Repositories[0].BaseBranch, "global value preserved separately")
}

func TestSetRepoBase_RequiresBranch(t *testing.T) {
	wm, _, _ := newSetBaseTestWorkspace(t, "repo1")
	err := wm.SetRepoBase(context.Background(), "ws", SetRepoBaseOptions{RepoName: "repo1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base branch is required")
}

func TestSetRepoBase_UnknownRepoErrors(t *testing.T) {
	wm, _, _ := newSetBaseTestWorkspace(t, "repo1")
	err := wm.SetRepoBase(context.Background(), "ws", SetRepoBaseOptions{RepoName: "nope", Branch: "main"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
