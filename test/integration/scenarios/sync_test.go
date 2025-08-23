package scenarios

import (
    "testing"

    h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestSyncAheadBehind(t *testing.T) {
    s := h.NewSandbox(t)
    defer s.Cleanup()
    s.SetBackend("hybrid")
    remote := s.InitBareRepo(t, "remote")
    _ = s.InitRepo(t, "repo1", remote)

    // Discover and create workspace with a single repo
    _ = s.RunWSM(t, nil, s.ReposDir, "discover", "--recursive=false")
    wsName := "ws-sync"
    res := s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1", "--branch", "feature/sync")
    if res.ExitCode != 0 { t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr) }
    wsPath := s.LoadWorkspacePath(t, wsName)

    // Diverge local: add one commit locally
    h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "echo local >> l.txt && git add l.txt && git commit -m local")

    // Diverge remote: push a commit directly from bare by cloning temp, or simpler: push from another clone
    other := s.InitRepo(t, "other", remote)
    h.RunForTest(t, s, other, nil, "bash", "-lc", "git checkout main && echo r >> r.txt && git add r.txt && git commit -m remote && git push")

    // Dry-run first
    res = s.RunWSM(t, nil, wsPath, "sync", "all", "--dry-run")
    if res.ExitCode != 0 { t.Fatalf("sync dry-run failed: %s\n%s", res.Stdout, res.Stderr) }

    // Pull then push
    res = s.RunWSM(t, nil, wsPath, "sync", "pull", "--rebase")
    if res.ExitCode != 0 { t.Fatalf("sync pull failed: %s\n%s", res.Stdout, res.Stderr) }
    res = s.RunWSM(t, nil, wsPath, "sync", "push")
    if res.ExitCode != 0 { t.Fatalf("sync push failed: %s\n%s", res.Stdout, res.Stderr) }
}


