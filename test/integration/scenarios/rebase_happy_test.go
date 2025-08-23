package scenarios

import (
    "testing"

    h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestRebaseHappyPath(t *testing.T) {
    s := h.NewSandbox(t)
    defer s.Cleanup()
    s.SetBackend("hybrid")
    remote := s.InitBareRepo(t, "remote")
    r1 := s.InitRepo(t, "repo1", remote)

    _ = s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
    wsName := "ws-rebase"
    res := s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1", "--branch", "feature/rb")
    if res.ExitCode != 0 { t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr) }
    wsPath := s.LoadWorkspacePath(t, wsName)

    // Add a commit on remote main
    other := s.InitRepo(t, "other-rb", remote)
    h.RunForTest(t, s, other, nil, "bash", "-lc", "git checkout main && echo r >> z && git add z && git commit -m r && git push")

    // Add local commits on feature branch in worktree
    h.RunForTest(t, s, r1, nil, "bash", "-lc", "git fetch && git checkout -B feature/rb origin/main")
    h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "echo l >> l && git add l && git commit -m l1")

    // Pull with rebase to linearize
    res = s.RunWSM(t, nil, wsPath, "sync", "pull", "--rebase")
    if res.ExitCode != 0 { t.Fatalf("rebase pull failed: %s\n%s", res.Stdout, res.Stderr) }
}


