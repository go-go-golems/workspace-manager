package scenarios

import (
    "os"
    "path/filepath"
    "testing"

    h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestWorktreeCreateDelete(t *testing.T) {
    s := h.NewSandbox(t)
    defer s.Cleanup()
    s.SetBackend("hybrid")
    remote := s.InitBareRepo(t, "remote")
    _ = s.InitRepo(t, "repo1", remote)
    _ = s.InitRepo(t, "repo2", remote)

    _ = s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
    wsName := "ws-wt"
    res := s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1,repo2", "--branch", "feature/wt")
    if res.ExitCode != 0 { t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr) }
    wsPath := s.LoadWorkspacePath(t, wsName)

    // Verify worktree directories exist
    for _, r := range []string{"repo1", "repo2"} {
        if _, err := os.Stat(filepath.Join(wsPath, r)); err != nil { t.Fatalf("worktree missing: %s", r) }
    }

    // Delete workspace and verify removal
    res = s.RunWSM(t, nil, wsPath, "delete", wsName, "--force", "--remove-files", "--force-worktrees")
    if res.ExitCode != 0 { t.Fatalf("delete failed: %s\n%s", res.Stdout, res.Stderr) }

    if _, err := os.Stat(wsPath); err == nil {
        t.Fatalf("workspace directory still exists: %s", wsPath)
    }
}


