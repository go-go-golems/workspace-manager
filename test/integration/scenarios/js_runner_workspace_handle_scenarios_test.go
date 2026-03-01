package scenarios

import (
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestJSRunnerWorkspaceHandleScripts(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)
	_ = s.InitRepo(t, "repo2", remote)

	discover := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if discover.ExitCode != 0 {
		t.Fatalf("discover failed:\nstdout:\n%s\nstderr:\n%s", discover.Stdout, discover.Stderr)
	}

	created := s.RunWSM(t, nil, "", "create", "ws-js-handle", "--repos", "repo1,repo2", "--branch", "feature/js-handle")
	if created.ExitCode != 0 {
		t.Fatalf("create ws-js-handle failed:\nstdout:\n%s\nstderr:\n%s", created.Stdout, created.Stderr)
	}

	basics := runRunnerScriptResult(t, s, "", "test/js/20-workspace-handle-basics.js")
	if name, _ := basics["name"].(string); name != "ws-js-handle" {
		t.Fatalf("unexpected handle basics result: %#v", basics)
	}

	gitResult := runRunnerScriptResult(t, s, "", "test/js/21-workspace-handle-git.js")
	if rowCount := asInt(t, gitResult["rebaseRowCount"]); rowCount < 1 {
		t.Fatalf("expected rebaseRowCount>=1, got %#v", gitResult)
	}

	parity := runRunnerScriptResult(t, s, "", "test/js/22-flat-vs-namespace-parity-extended.js")
	parityMap, ok := parity["parity"].(map[string]any)
	if !ok {
		t.Fatalf("expected parity map, got %#v", parity)
	}
	for key, value := range parityMap {
		okVal, _ := value.(bool)
		if !okVal {
			t.Fatalf("expected parity[%s]=true, got %#v", key, parityMap)
		}
	}
}
