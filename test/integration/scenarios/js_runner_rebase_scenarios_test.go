package scenarios

import (
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestJSRunnerRebaseScriptsHappyPath(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()
	s.SetBackend("hybrid")

	prepareJSRebaseWorkspace(t, s)

	run := runRunnerScriptResult(t, s, "", "test/js/16-git-rebase-run-happy.js")
	if rowCount := asInt(t, run["rowCount"]); rowCount != 1 {
		t.Fatalf("expected one rebase row, got %#v", run)
	}

	status := runRunnerScriptResult(t, s, "", "test/js/17-git-rebase-status.js")
	if rowCount := asInt(t, status["rowCount"]); rowCount != 1 {
		t.Fatalf("expected one rebase status row, got %#v", status)
	}
}

func TestJSRunnerRebaseScriptsConflictFlow(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()
	s.SetBackend("hybrid")

	remote := s.InitBareRepo(t, "remote")
	repo1 := s.InitRepo(t, "repo1", remote)

	h.RunForTest(t, s, repo1, nil, "bash", "-lc", "git checkout main && printf 'a\\nb\\nc\\n' > f.txt && git add f.txt && git commit -m base && git push")

	discover := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if discover.ExitCode != 0 {
		t.Fatalf("discover failed:\nstdout:\n%s\nstderr:\n%s", discover.Stdout, discover.Stderr)
	}

	created := s.RunWSM(t, nil, "", "create", "ws-js-rebase", "--repos", "repo1", "--branch", "feature/js-rebase")
	if created.ExitCode != 0 {
		t.Fatalf("create ws-js-rebase failed:\nstdout:\n%s\nstderr:\n%s", created.Stdout, created.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, "ws-js-rebase")

	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "sed -i 's/^b$/local/' f.txt && git add f.txt && git commit -m local")

	other := s.InitRepo(t, "other-js-rebase", remote)
	h.RunForTest(t, s, other, nil, "bash", "-lc", "git checkout main && sed -i 's/^b$/remote/' f.txt && git add f.txt && git commit -m remote && git push")

	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "git fetch origin && git rebase origin/main || true")

	status := runRunnerScriptResult(t, s, "", "test/js/17-git-rebase-status.js")
	if rowCount := asInt(t, status["rowCount"]); rowCount != 1 {
		t.Fatalf("expected one status row, got %#v", status)
	}

	continueResult := runRunnerScriptResult(t, s, "", "test/js/18-git-rebase-continue.js")
	if rowCount := asInt(t, continueResult["rowCount"]); rowCount != 1 {
		t.Fatalf("expected one continue row, got %#v", continueResult)
	}

	abortResult := runRunnerScriptResult(t, s, "", "test/js/19-git-rebase-abort.js")
	if rowCount := asInt(t, abortResult["rowCount"]); rowCount != 1 {
		t.Fatalf("expected one abort row, got %#v", abortResult)
	}
}

func prepareJSRebaseWorkspace(t *testing.T, s *h.Sandbox) {
	t.Helper()

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)

	discover := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if discover.ExitCode != 0 {
		t.Fatalf("discover failed:\nstdout:\n%s\nstderr:\n%s", discover.Stdout, discover.Stderr)
	}

	created := s.RunWSM(t, nil, "", "create", "ws-js-rebase", "--repos", "repo1", "--branch", "feature/js-rebase")
	if created.ExitCode != 0 {
		t.Fatalf("create ws-js-rebase failed:\nstdout:\n%s\nstderr:\n%s", created.Stdout, created.Stderr)
	}
}
