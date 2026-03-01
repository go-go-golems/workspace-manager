package scenarios

import (
	"encoding/json"
	"strings"
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestBranchAndLogHumanDataParity(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)
	_ = s.InitRepo(t, "repo2", remote)

	res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if res.ExitCode != 0 {
		t.Fatalf("discover failed: %s\n%s", res.Stdout, res.Stderr)
	}

	wsName := "ws-branch-log"
	res = s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1,repo2", "--branch", "feature/branch-log")
	if res.ExitCode != 0 {
		t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, wsName)

	// Ensure a recent commit exists in the workspace branch for log assertions.
	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "echo log-check >> branch-log.txt && git add branch-log.txt && git commit -m 'branch log parity'")

	// branch list: human
	res = s.RunWSM(t, nil, wsPath, "branch", "list")
	if res.ExitCode != 0 {
		t.Fatalf("branch list failed: %s\n%s", res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "repo1") || !strings.Contains(res.Stdout, "repo2") {
		t.Fatalf("branch list human output missing repositories: %s", res.Stdout)
	}

	// branch list: data
	res = s.RunWSM(t, nil, wsPath, "branch", "list", "--with-glaze-output", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("branch list data failed: %s\n%s", res.Stdout, res.Stderr)
	}
	branchRows := parseJSONRows(t, res.Stdout)
	if len(branchRows) < 2 {
		t.Fatalf("expected at least 2 branch rows, got %d: %s", len(branchRows), res.Stdout)
	}

	// log: human
	res = s.RunWSM(t, nil, wsPath, "log", "--limit", "5", "--oneline")
	if res.ExitCode != 0 {
		t.Fatalf("log failed: %s\n%s", res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "Commit history for workspace") {
		t.Fatalf("unexpected human log output: %s", res.Stdout)
	}

	// log: data
	res = s.RunWSM(t, nil, wsPath, "log", "--limit", "5", "--oneline", "--with-glaze-output", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("log data failed: %s\n%s", res.Stdout, res.Stderr)
	}
	logRows := parseJSONRows(t, res.Stdout)
	if len(logRows) == 0 {
		t.Fatalf("expected at least one log row, got none: %s", res.Stdout)
	}
}

func parseJSONRows(t *testing.T, stdout string) []map[string]any {
	t.Helper()
	var rows []map[string]any
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nstdout:\n%s", err, stdout)
	}
	return rows
}
