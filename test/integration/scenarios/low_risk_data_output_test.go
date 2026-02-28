package scenarios

import (
	"encoding/json"
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestLowRiskCommandsDataOutput(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()
	s.SetBackend("hybrid")

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)
	_ = s.InitRepo(t, "repo2", remote)

	res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if res.ExitCode != 0 {
		t.Fatalf("discover failed: %s\n%s", res.Stdout, res.Stderr)
	}

	wsName := "ws-low-risk"
	res = s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1,repo2", "--branch", "feature/low-risk")
	if res.ExitCode != 0 {
		t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr)
	}

	// list repos (data mode)
	res = s.RunWSM(t, nil, "", "list", "repos", "--output-mode", "data", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("list repos data failed: %s\n%s", res.Stdout, res.Stderr)
	}
	repoRows := parseDataRows(t, res.Stdout)
	if len(repoRows) < 2 {
		t.Fatalf("expected at least 2 repo rows, got %d: %s", len(repoRows), res.Stdout)
	}

	// list workspaces (data mode)
	res = s.RunWSM(t, nil, "", "list", "workspaces", "--output-mode", "data", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("list workspaces data failed: %s\n%s", res.Stdout, res.Stderr)
	}
	workspaceRows := parseDataRows(t, res.Stdout)
	if len(workspaceRows) == 0 {
		t.Fatalf("expected at least one workspace row, got none: %s", res.Stdout)
	}

	// info (data mode)
	res = s.RunWSM(t, nil, "", "info", wsName, "--output-mode", "data", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("info data failed: %s\n%s", res.Stdout, res.Stderr)
	}
	infoRows := parseDataRows(t, res.Stdout)
	if len(infoRows) != 1 {
		t.Fatalf("expected exactly one info row, got %d: %s", len(infoRows), res.Stdout)
	}
	workspace, ok := infoRows[0]["name"].(string)
	if !ok || workspace != wsName {
		t.Fatalf("unexpected workspace in info row: %#v", infoRows[0]["workspace"])
	}
}

func parseDataRows(t *testing.T, stdout string) []map[string]any {
	t.Helper()
	var rows []map[string]any
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nstdout:\n%s", err, stdout)
	}
	return rows
}
