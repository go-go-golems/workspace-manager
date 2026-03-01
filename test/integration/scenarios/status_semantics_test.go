package scenarios

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func parseStatusRows(t *testing.T, stdout string) []map[string]any {
	t.Helper()

	jsonPart := stdout
	if idx := strings.LastIndex(stdout, "\n["); idx >= 0 {
		jsonPart = stdout[idx+1:]
	} else if idx := strings.Index(stdout, "["); idx >= 0 {
		jsonPart = stdout[idx:]
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonPart), &rows); err != nil {
		t.Fatalf("failed to parse status JSON rows: %v\nstdout:\n%s", err, stdout)
	}
	return rows
}

func TestStatusSemanticMergedAndNeedsRebase(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)

	res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if res.ExitCode != 0 {
		t.Fatalf("discover failed: %s\n%s", res.Stdout, res.Stderr)
	}

	wsName := "ws-status-semantics"
	res = s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1", "--branch", "feature/semantic")
	if res.ExitCode != 0 {
		t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, wsName)
	repoInWorkspace := filepath.Join(wsPath, "repo1")

	// Add one feature commit so the branch is not merged.
	h.RunForTest(t, s, repoInWorkspace, nil, "bash", "-lc", "echo feature >> feature.txt && git add feature.txt && git commit -m 'feature change'")

	// Advance remote main independently.
	seed := s.InitRepo(t, "seed", remote)
	h.RunForTest(t, s, seed, nil, "bash", "-lc", "git checkout main && echo remote >> README.md && git add README.md && git commit -m 'remote main update' && git push origin main")

	// Without --fetch, status should use stale remote-tracking refs.
	res = s.RunWSM(t, nil, wsPath, "status", "--workspace", wsName, "--with-glaze-output", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("status failed: %s\n%s", res.Stdout, res.Stderr)
	}

	rows := parseStatusRows(t, res.Stdout)
	if len(rows) != 1 {
		t.Fatalf("expected one status row, got %d: %s", len(rows), res.Stdout)
	}
	row := rows[0]

	merged, _ := row["is_merged"].(bool)
	if merged {
		t.Fatalf("expected is_merged=false, row=%#v", row)
	}
	needsRebase, _ := row["needs_rebase"].(bool)
	if needsRebase {
		t.Fatalf("expected needs_rebase=false without --fetch (stale refs), row=%#v", row)
	}

	// With --fetch, status should update origin refs and detect that rebase is needed.
	res = s.RunWSM(t, nil, wsPath, "status", "--workspace", wsName, "--fetch", "--with-glaze-output", "--output", "json")
	if res.ExitCode != 0 {
		t.Fatalf("status --fetch failed: %s\n%s", res.Stdout, res.Stderr)
	}
	rows = parseStatusRows(t, res.Stdout)
	if len(rows) != 1 {
		t.Fatalf("expected one status row after --fetch, got %d: %s", len(rows), res.Stdout)
	}
	row = rows[0]
	needsRebase, _ = row["needs_rebase"].(bool)
	if !needsRebase {
		t.Fatalf("expected needs_rebase=true after --fetch updates refs, row=%#v", row)
	}
}
