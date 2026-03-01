package scenarios

import (
	"strings"
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

// TestRebaseConflictsAbort verifies that aborting a conflicted rebase rolls back state.
func TestRebaseConflictsAbort(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()
	s.SetBackend("hybrid")
	remote := s.InitBareRepo(t, "remote")
	r1 := s.InitRepo(t, "repo1", remote)

	// Base commit
	h.RunForTest(t, s, r1, nil, "bash", "-lc", "git checkout main && printf 'a\\nb\\nc\\n' > f.txt && git add f.txt && git commit -m base && git push")

	// Create workspace on feature
	res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	t.Logf("[debug] discover exit=%d\nstdout:\n%s\nstderr:\n%s", res.ExitCode, res.Stdout, res.Stderr)
	wsName := "ws-rbc-abort"
	res = s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1", "--branch", "feature/rbc")
	if res.ExitCode != 0 {
		t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, wsName)

	// Commit on feature in worktree
	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "sed -i 's/^b$/local/' f.txt && git add f.txt && git commit -m local")

	// Remote changes on main
	other := s.InitRepo(t, "other-rbc-abort", remote)
	h.RunForTest(t, s, other, nil, "bash", "-lc", "git checkout main && sed -i 's/^b$/remote/' f.txt && git add f.txt && git commit -m remote && git push")

	// Start rebase to create conflict
	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "git fetch origin && git rebase origin/main || true")

	// Verify conflicts via status or conflicts list
	st := s.RunWSM(t, nil, wsPath, "rebase", "status", "--repo", "repo1")
	t.Logf("[debug] rebase status (pre-abort):\n%s", st.Stdout)
	hasStopped := strings.Contains(st.Stdout, "stopped-conflicts") || strings.Contains(st.Stdout, "\trepo1\t")
	if !hasStopped {
		cl := s.RunWSM(t, nil, wsPath, "conflicts", "list", "--repo", "repo1")
		if !strings.Contains(cl.Stdout, "repo1") || !strings.Contains(cl.Stdout, "1") {
			t.Fatalf("expected conflicts before abort; status: %s\nconflicts: %s", st.Stdout, cl.Stdout)
		}
	}

	// Abort
	res = s.RunWSM(t, nil, wsPath+"/repo1", "rebase", "abort")
	if res.ExitCode != 0 {
		t.Fatalf("rebase abort failed: %s\n%s", res.Stdout, res.Stderr)
	}

	// State should be clean or at least no conflicts
	st2 := s.RunWSM(t, nil, wsPath, "rebase", "status", "--repo", "repo1")
	t.Logf("[debug] rebase status (post-abort):\n%s", st2.Stdout)
	if strings.Contains(st2.Stdout, "stopped-conflicts") {
		t.Fatalf("expected no conflicts after abort, got: %s", st2.Stdout)
	}
}
