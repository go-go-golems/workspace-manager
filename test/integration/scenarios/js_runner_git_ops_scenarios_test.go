package scenarios

import (
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestJSRunnerGitOpsScripts(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()
	s.SetBackend("hybrid")

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)
	_ = s.InitRepo(t, "repo2", remote)

	discover := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if discover.ExitCode != 0 {
		t.Fatalf("discover failed:\nstdout:\n%s\nstderr:\n%s", discover.Stdout, discover.Stderr)
	}

	created := s.RunWSM(t, nil, "", "create", "ws-js-git", "--repos", "repo1,repo2", "--branch", "feature/js-git")
	if created.ExitCode != 0 {
		t.Fatalf("create ws-js-git failed:\nstdout:\n%s\nstderr:\n%s", created.Stdout, created.Stderr)
	}

	wsPath := s.LoadWorkspacePath(t, "ws-js-git")
	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "echo x >> README.md")
	h.RunForTest(t, s, wsPath+"/repo2", nil, "bash", "-lc", "echo y >> README.md")

	commitResult := runRunnerScriptResult(t, s, "", "test/js/12-git-commit.js")
	if status, _ := commitResult["status"].(string); status == "" {
		t.Fatalf("expected commit status string, got %#v", commitResult)
	}

	diffResult := runRunnerScriptResult(t, s, "", "test/js/13-git-diff.js")
	if parity, _ := diffResult["parity"].(bool); !parity {
		t.Fatalf("expected diff parity=true, got %#v", diffResult)
	}

	logResult := runRunnerScriptResult(t, s, "", "test/js/14-git-log.js")
	if repoCount := asInt(t, logResult["repoCountFlat"]); repoCount < 1 {
		t.Fatalf("expected repoCountFlat>=1, got %#v", logResult)
	}

	branchResult := runRunnerScriptResult(t, s, "", "test/js/15-git-branch-create-switch-list.js")
	if listCount := asInt(t, branchResult["listEntryCount"]); listCount < 1 {
		t.Fatalf("expected listEntryCount>=1, got %#v", branchResult)
	}
}
