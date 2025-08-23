package scenarios

import (
    "strings"
    "testing"

    h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestCommitPush(t *testing.T) {
    s := h.NewSandbox(t)
    defer s.Cleanup()
    s.SetBackend("hybrid")
    remote := s.InitBareRepo(t, "remote")
    _ = s.InitRepo(t, "repo1", remote)
    _ = s.InitRepo(t, "repo2", remote)

    // Discover and create workspace
    _ = s.RunWSM(t, nil, s.ReposDir, "discover", "--recursive=false")
    wsName := "ws2"
    res := s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1,repo2", "--branch", "feature/commit")
    if res.ExitCode != 0 { t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr) }

    // Modify files in both, then commit+push
    wsPath := s.LoadWorkspacePath(t, wsName)
    h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "echo x >> a.txt && git add a.txt")
    h.RunForTest(t, s, wsPath+"/repo2", nil, "bash", "-lc", "echo y >> b.txt && git add b.txt")

    res = s.RunWSM(t, nil, wsPath, "commit", "-m", "test commit", "--add-all", "--push")
    if res.ExitCode != 0 { t.Fatalf("commit failed: %s\n%s", res.Stdout, res.Stderr) }

    // Verify status clean
    res = s.RunWSM(t, nil, wsPath, "status", "--workspace", wsName)
    if res.ExitCode != 0 { t.Fatalf("status failed: %s\n%s", res.Stdout, res.Stderr) }
    if strings.Contains(res.Stdout, "Modified files:") {
        t.Fatalf("expected clean status, got: %s", res.Stdout)
    }
}


