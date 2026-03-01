package scenarios

import (
	"strconv"
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

	// Deterministic same-line conflict setup on main/feature
	// Base commit on main
	h.RunForTest(t, s, r1, nil, "bash", "-lc", "git checkout main && printf 'a\\nb\\nc\\n' > f.txt && git add f.txt && git commit -m base && git push")

	// Discover repos and create workspace worktree on feature branch from current origin/main
	t.Logf("[debug] running discover in %s", s.ReposDir)
	res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	t.Logf("[debug] discover exit=%d\nstdout:\n%s\nstderr:\n%s", res.ExitCode, res.Stdout, res.Stderr)
	wsName := "ws-rbc"
	res = s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1", "--branch", "feature/rbc")
	if res.ExitCode != 0 {
		t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, wsName)

	// Local edit on feature branch inside worktree (pre-remote change)
	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "sed -i 's/^b$/local/' f.txt && git add f.txt && git commit -m local")

	// Separate clone modifies main with 'b' -> 'remote' and pushes
	other := s.InitRepo(t, "other-rbc", remote)
	h.RunForTest(t, s, other, nil, "bash", "-lc", "git checkout main && sed -i 's/^b$/remote/' f.txt && git add f.txt && git commit -m remote && git push")

	// Perform raw git rebase inside the worktree to ensure a conflict stops the rebase
	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "git fetch origin && git rebase origin/main || true")

	// Debug: show upstream, rebase dirs, porcelain
	up := h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "git rev-parse --abbrev-ref @{upstream} || true")
	t.Logf("[debug] upstream: %s", strings.TrimSpace(up))
	rebaseDirs := h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "(ls .git | grep -E 'rebase-(merge|apply)' || true)")
	t.Logf("[debug] rebase-dirs: %q", strings.TrimSpace(rebaseDirs))
	porcelain := h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "git status --porcelain || true")
	t.Logf("[debug] porcelain:\n%s", porcelain)

	// Check rebase status should show stopped-conflicts OR conflicts present via conflicts list
	res = s.RunWSM(t, nil, wsPath, "rebase", "status", "--repo", "repo1")
	t.Logf("[debug] rebase status:\n%s", res.Stdout)
	hasStopped := strings.Contains(res.Stdout, "stopped-conflicts")
	if !hasStopped {
		// Fallback: confirm conflicts exist via conflicts list
		cl := s.RunWSM(t, nil, wsPath, "conflicts", "list", "--repo", "repo1")
		if cl.ExitCode != 0 {
			t.Fatalf("conflicts list failed: %s\n%s", cl.Stdout, cl.Stderr)
		}
		t.Logf("[debug] conflicts list:\n%s", cl.Stdout)
		// Parse the table row for repo1 and verify count > 0 (tabs may be expanded to spaces)
		lines := strings.Split(strings.TrimSpace(cl.Stdout), "\n")
		found := false
		for _, ln := range lines {
			if strings.HasPrefix(strings.TrimSpace(ln), "repo1") {
				fields := strings.Fields(ln)
				if len(fields) >= 2 {
					n, err := strconv.Atoi(fields[1])
					if err == nil && n > 0 {
						found = true
						break
					}
				}
			}
		}
		if !found {
			t.Fatalf("expected stopped-conflicts or conflicts>0; status: %s\nconflicts list: %s", res.Stdout, cl.Stdout)
		}
	}

	// Mark all resolved by taking local and continue
	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "echo resolved > f.txt && git add f.txt")
	res = s.RunWSM(t, nil, wsPath+"/repo1", "rebase", "continue")
	if res.ExitCode != 0 {
		t.Fatalf("rebase continue failed: %s\n%s", res.Stdout, res.Stderr)
	}

	// Optional: Abort path covered in separate test
}
