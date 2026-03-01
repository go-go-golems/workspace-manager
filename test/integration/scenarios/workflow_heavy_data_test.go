package scenarios

import (
	"encoding/json"
	"strings"
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestWorkflowHeavyCommandsDataOutput(t *testing.T) {
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

	// create (data mode)
	res = s.RunWSM(t, nil, "", "create", "ws-heavy-source", "--repos", "repo1,repo2", "--branch", "feature/heavy", "--with-glaze-output", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("create data failed: %s\n%s", res.Stdout, res.Stderr)
	}
	createRows := parseHeavyRows(t, res.Stdout)
	if len(createRows) != 1 {
		t.Fatalf("expected one create row, got %d: %s", len(createRows), res.Stdout)
	}

	wsPath := s.LoadWorkspacePath(t, "ws-heavy-source")
	h.RunForTest(t, s, wsPath+"/repo1", nil, "bash", "-lc", "echo heavy-data >> heavy.txt")
	h.RunForTest(t, s, wsPath+"/repo2", nil, "bash", "-lc", "echo heavy-data >> heavy.txt")

	// commit (data mode)
	res = s.RunWSM(t, nil, wsPath, "commit", "--add-all", "-m", "workflow heavy data", "--with-glaze-output", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("commit data failed: %s\n%s", res.Stdout, res.Stderr)
	}
	commitRows := parseHeavyRows(t, res.Stdout)
	if len(commitRows) == 0 {
		t.Fatalf("expected commit rows, got none: %s", res.Stdout)
	}

	// delete (data mode)
	res = s.RunWSM(t, nil, "", "delete", "ws-heavy-source", "--force", "--remove-files", "--force-worktrees", "--with-glaze-output", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("delete data failed: %s\n%s", res.Stdout, res.Stderr)
	}
	deleteRows := parseHeavyRows(t, res.Stdout)
	if len(deleteRows) != 1 {
		t.Fatalf("expected one delete row, got %d: %s", len(deleteRows), res.Stdout)
	}
	if deleteRows[0]["status"] != "deleted" {
		t.Fatalf("unexpected delete status: %#v", deleteRows[0]["status"])
	}
}

func parseHeavyRows(t *testing.T, stdout string) []map[string]any {
	t.Helper()
	jsonPart := stdout
	if idx := strings.LastIndex(stdout, "\n["); idx >= 0 {
		jsonPart = stdout[idx+1:]
	} else if idx := strings.Index(stdout, "["); idx >= 0 {
		jsonPart = stdout[idx:]
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonPart), &rows); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nstdout:\n%s", err, stdout)
	}
	return rows
}
