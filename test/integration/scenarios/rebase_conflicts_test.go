package scenarios

import (
    "strings"
    "testing"

    h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestRebaseConflictsContinueAbort(t *testing.T) {
    s := h.NewSandbox(t)
    defer s.Cleanup()
    s.SetBackend("hybrid")
    remote := s.InitBareRepo(t, "remote")
    r1 := s.InitRepo(t, "repo1", remote)

    _ = s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
    wsName := "ws-rbc"
    res := s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1", "--branch", "feature/rbc")
    if res.ExitCode != 0 { t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr) }
    wsPath := s.LoadWorkspacePath(t, wsName)

    // Create divergence with conflicting changes to the same file
    h.RunForTest(t, s, r1, nil, "bash", "-lc", "git checkout main && echo base > f.txt && git add f.txt && git commit -m base && git push")
    h.RunForTest(t, s, r1, nil, "bash", "-lc", "git checkout -B feature/rbc origin/main && echo local1 > f.txt && git add f.txt && git commit -m local1")
    other := s.InitRepo(t, "other-rbc", remote)
    h.RunForTest(t, s, other, nil, "bash", "-lc", "git checkout main && echo remote1 > f.txt && git add f.txt && git commit -m remote1 && git push")

    // Pull with rebase to trigger conflicts
    res = s.RunWSM(t, nil, wsPath, "sync", "pull", "--rebase")
    if res.ExitCode == 0 {
        t.Fatalf("expected conflicts during rebase, got success: %s", res.Stdout)
    }

    // Check rebase status (should show stopped-conflicts)
    res = s.RunWSM(t, nil, wsPath, "rebase", "status", "--repo", "repo1")
    if res.ExitCode != 0 || !strings.Contains(res.Stdout, "stopped-conflicts") {
        t.Fatalf("expected stopped-conflicts, got: %s\n%s", res.Stdout, res.Stderr)
    }

    // Mark all resolved by taking local and continue
    h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "echo resolved > f.txt && git add f.txt")
    res = s.RunWSM(t, nil, wsPath+"/repo1", "rebase", "continue")
    // continue is part of rebase root subcommand; it expects cwd inside repo; use direct helper
    if res.ExitCode != 0 {
        t.Fatalf("rebase continue failed: %s\n%s", res.Stdout, res.Stderr)
    }
}


