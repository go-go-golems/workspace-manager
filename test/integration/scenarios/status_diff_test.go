package scenarios

import (
    "path/filepath"
    "strings"
    "testing"

    h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

// TestSmokeStatusDiff creates two repos with a bare remote, creates a workspace from them,
// then runs `wsm status` and `wsm diff` to ensure basic outputs are produced.
func TestSmokeStatusDiff(t *testing.T) {
    s := h.NewSandbox(t)
    defer s.Cleanup()
    s.SetBackend("hybrid")

    // Setup bare remote and two local repos
    remote := s.InitBareRepo(t, "remote")
    _ = s.InitRepo(t, "repo1", remote)
    _ = s.InitRepo(t, "repo2", remote)

    // Discover repos into registry limited to our sandbox repos dir
    // We run discovery pointing to s.ReposDir so WSM can find repo names 'repo1', 'repo2'
    t.Logf("[debug] running discover in %s", s.ReposDir)
    res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
    t.Logf("[debug] discover exit=%d\nstdout:\n%s\nstderr:\n%s", res.ExitCode, res.Stdout, res.Stderr)

    // Create workspace using those repos
    wsName := "ws1"
    res = s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1,repo2", "--branch", "feature/x")
    if res.ExitCode != 0 {
        t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr)
    }

    // Retrieve workspace path
    wsPath := s.LoadWorkspacePath(t, wsName)
    if wsPath == "" {
        t.Fatalf("workspace path not found")
    }

    // Make a modification to produce diff
    h.RunForTest(t, s, filepath.Join(wsPath, "repo1"), nil, "bash", "-lc", "echo change >> file.txt && git add file.txt && git commit -m change")

    // status
    res = s.RunWSM(t, nil, wsPath, "status", "--workspace", wsName)
    if res.ExitCode != 0 { t.Fatalf("status failed: %s\n%s", res.Stdout, res.Stderr) }
    if !strings.Contains(res.Stdout, "Workspace:") && !strings.Contains(res.Stdout, "Overall Status") {
        t.Fatalf("unexpected status output: %s", res.Stdout)
    }

    // diff
    res = s.RunWSM(t, nil, wsPath, "diff")
    if res.ExitCode != 0 { t.Fatalf("diff failed: %s\n%s", res.Stdout, res.Stderr) }
    if !strings.Contains(res.Stdout, "=== Repository: repo1 ===") && !strings.Contains(res.Stdout, "No changes") {
        t.Fatalf("unexpected diff output: %s", res.Stdout)
    }
}


