package scenarios

import (
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestJSRunnerWorkspaceLifecycleScripts(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()
	s.SetBackend("hybrid")

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)
	repo2 := s.InitRepo(t, "repo2", remote)
	h.RunForTest(t, s, repo2, nil, "bash", "-lc", "git checkout -b feature/js-ops-repo2 && git push -u origin feature/js-ops-repo2 && git checkout main")

	discover := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if discover.ExitCode != 0 {
		t.Fatalf("discover failed:\nstdout:\n%s\nstderr:\n%s", discover.Stdout, discover.Stderr)
	}

	created := s.RunWSM(t, nil, "", "create", "ws-js-ops", "--repos", "repo1", "--branch", "feature/js-ops")
	if created.ExitCode != 0 {
		t.Fatalf("create ws-js-ops failed:\nstdout:\n%s\nstderr:\n%s", created.Stdout, created.Stderr)
	}

	info := runRunnerScriptResult(t, s, "", "test/js/08-workspace-info.js")
	if workspace, _ := info["workspace"].(string); workspace != "ws-js-ops" {
		t.Fatalf("unexpected workspace info result: %#v", info)
	}

	forkMerge := runRunnerScriptResult(t, s, "", "test/js/11-workspace-fork-merge.js")
	forkSucceeded, _ := forkMerge["forkSucceeded"].(bool)
	if !forkSucceeded {
		forkError, _ := forkMerge["forkError"].(string)
		if forkError == "" {
			t.Fatalf("expected fork error details when fork is unsuccessful, got %#v", forkMerge)
		}
	}

	addRemove := runRunnerScriptResult(t, s, "", "test/js/09-workspace-add-remove.js")
	if afterAdd := asInt(t, addRemove["repositoryCountAfterAdd"]); afterAdd != 2 {
		t.Fatalf("expected repositoryCountAfterAdd=2, got %#v", addRemove)
	}
	if afterRemove := asInt(t, addRemove["repositoryCountAfterRemove"]); afterRemove != 1 {
		t.Fatalf("expected repositoryCountAfterRemove=1, got %#v", addRemove)
	}

	deleted := runRunnerScriptResult(t, s, "", "test/js/10-workspace-delete.js")
	if presentAfterDelete, _ := deleted["workspacePresentAfterDelete"].(bool); presentAfterDelete {
		t.Fatalf("workspace should not be present after delete: %#v", deleted)
	}
}
