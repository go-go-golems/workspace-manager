package scenarios

import (
    "testing"

    h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestJobsConcurrency(t *testing.T) {
    s := h.NewSandbox(t)
    defer s.Cleanup()
    s.SetBackend("hybrid")
    remote := s.InitBareRepo(t, "remote")
    _ = s.InitRepo(t, "repo1", remote)
    _ = s.InitRepo(t, "repo2", remote)
    _ = s.InitRepo(t, "repo3", remote)

    _ = s.RunWSM(t, nil, s.ReposDir, "discover", "--recursive=false")
    wsName := "ws-jobs"
    res := s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1,repo2,repo3", "--branch", "feature/jobs")
    if res.ExitCode != 0 { t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr) }
    wsPath := s.LoadWorkspacePath(t, wsName)

    res = s.RunWSM(t, nil, wsPath, "status", "--workspace", wsName, "--jobs", "4")
    if res.ExitCode != 0 { t.Fatalf("status with jobs failed: %s\n%s", res.Stdout, res.Stderr) }
    res = s.RunWSM(t, nil, wsPath, "diff", "--jobs", "4")
    if res.ExitCode != 0 { t.Fatalf("diff with jobs failed: %s\n%s", res.Stdout, res.Stderr) }
}


