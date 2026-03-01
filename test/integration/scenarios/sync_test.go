package scenarios

import (
	"strings"
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestRebaseDataModeDryRun(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()
	s.SetBackend("hybrid")
	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)

	// Discover and create workspace with a single repo
	_ = s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	wsName := "ws-rebase-data"
	res := s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1", "--branch", "feature/rebase-data")
	if res.ExitCode != 0 {
		t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, wsName)

	// Diverge remote main and local feature to ensure rebase has work to do.
	other := s.InitRepo(t, "other", remote)
	h.RunForTest(t, s, other, nil, "bash", "-lc", "git checkout main && echo remote >> r.txt && git add r.txt && git commit -m remote && git push")
	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "echo local >> l.txt && git add l.txt && git commit -m local")

	// Dry-run in data mode should succeed and include structured rows.
	res = s.RunWSM(t, nil, wsPath, "rebase", "--target", "main", "--dry-run", "--with-glaze-output", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("rebase dry-run failed: %s\n%s", res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "\"repository\"") {
		t.Fatalf("expected repository field in data output: %s", res.Stdout)
	}
}
