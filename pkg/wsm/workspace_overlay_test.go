package wsm

import (
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
