package scenarios

import (
	"strings"
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestRebaseHappyPath(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()
	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)

	_ = s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	wsName := "ws-rebase"
	res := s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1", "--branch", "feature/rb")
	if res.ExitCode != 0 {
		t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, wsName)

	// Add a commit on remote main
	other := s.InitRepo(t, "other-rb", remote)
	h.RunForTest(t, s, other, nil, "bash", "-lc", "git checkout main && echo r >> z && git add z && git commit -m r && git push")

	// Add local commit on feature branch in worktree.
	// The branch is already checked out by the worktree created above; checking it out again
	// in the source repository fails because Git allows one checked-out worktree per branch.
	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "echo l >> l && git add l && git commit -m l1")

	// Rebase workspace branch onto updated main
	res = s.RunWSM(t, nil, wsPath, "rebase", "--target", "main")
	if res.ExitCode != 0 {
		t.Fatalf("rebase failed: %s\n%s", res.Stdout, res.Stderr)
	}

	// Ensure no conflict/stopped status remains
	status := s.RunWSM(t, nil, wsPath, "rebase", "status", "--repo", "repo1")
	if status.ExitCode != 0 {
		t.Fatalf("rebase status failed: %s\n%s", status.Stdout, status.Stderr)
	}
	if strings.Contains(status.Stdout, "stopped-conflicts") {
		t.Fatalf("unexpected stopped-conflicts in happy path:\n%s", status.Stdout)
	}
}
