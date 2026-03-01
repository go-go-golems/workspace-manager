package scenarios

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestJSRunnerDemoScriptsLifecycleScenario(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)
	_ = s.InitRepo(t, "repo2", remote)

	moduleSurface := runRunnerScriptResult(t, s, "", "test/js/00-module-surface.js")
	if moduleSurface["script"] != "00-module-surface" {
		t.Fatalf("unexpected script result: %#v", moduleSurface)
	}

	discovery := runRunnerScriptResult(t, s, s.ReposDir, "test/js/01-discover-and-list.js")
	if count := asInt(t, discovery["repositoryCountFromDiscover"]); count < 2 {
		t.Fatalf("expected discover count >= 2, got %d (%#v)", count, discovery)
	}
	if count := asInt(t, discovery["repositoryCountFromList"]); count < 2 {
		t.Fatalf("expected list count >= 2, got %d (%#v)", count, discovery)
	}

	listParity := runRunnerScriptResult(t, s, "", "test/js/05-list-repository-parity.js")
	if parity, _ := listParity["parity"].(bool); !parity {
		t.Fatalf("expected repository list parity, got %#v", listParity)
	}

	created := runRunnerScriptResult(t, s, "", "test/js/02-create-workspace.js")
	if workspace, _ := created["workspace"].(string); workspace != "ws-js-demo" {
		t.Fatalf("expected workspace ws-js-demo, got %#v", created)
	}
	if branch, _ := created["finalBranch"].(string); branch != "feature/js-demo" {
		t.Fatalf("expected finalBranch feature/js-demo, got %#v", created)
	}

	statusParity := runRunnerScriptResult(t, s, "", "test/js/03-status-namespace-parity.js")
	parityObj, ok := statusParity["parity"].(map[string]any)
	if !ok {
		t.Fatalf("expected parity object, got %#v", statusParity)
	}
	if rootVsWorkspaces, _ := parityObj["rootVsWorkspaces"].(bool); !rootVsWorkspaces {
		t.Fatalf("expected rootVsWorkspaces=true, got %#v", statusParity)
	}
	if rootVsGit, _ := parityObj["rootVsGit"].(bool); !rootVsGit {
		t.Fatalf("expected rootVsGit=true, got %#v", statusParity)
	}

	jobsStatus := runRunnerScriptResult(t, s, "", "test/js/07-default-jobs-status.js")
	if repoCount := asInt(t, jobsStatus["repositoryCount"]); repoCount != 2 {
		t.Fatalf("expected repositoryCount=2, got %#v", jobsStatus)
	}

	validation := runRunnerScriptResult(t, s, "", "test/js/06-validation-errors.js")
	missingName, _ := validation["missingNameMessage"].(string)
	missingRepos, _ := validation["missingReposMessage"].(string)
	if !strings.Contains(missingName, "createWorkspace requires name") {
		t.Fatalf("expected missing-name message, got %#v", validation)
	}
	if !strings.Contains(missingRepos, "createWorkspace requires repos") {
		t.Fatalf("expected missing-repos message, got %#v", validation)
	}
}

func TestJSRunnerDemoScriptsConvenienceScenario(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)

	result := runRunnerScriptResult(t, s, s.ReposDir, "test/js/04-convenience-lifecycle.js")
	if workspace, _ := result["workspace"].(string); workspace != "ws-js-convenience" {
		t.Fatalf("expected workspace ws-js-convenience, got %#v", result)
	}
	if branch, _ := result["finalBranch"].(string); branch != "feat/ws-js-convenience" {
		t.Fatalf("expected generated branch feat/ws-js-convenience, got %#v", result)
	}
	if repoCount := asInt(t, result["repositoryCount"]); repoCount != 1 {
		t.Fatalf("expected repositoryCount=1, got %#v", result)
	}
}

func runRunnerScriptResult(t *testing.T, s *h.Sandbox, workDir, scriptRelPath string) map[string]any {
	t.Helper()

	scriptPath := filepath.Join(projectRoot(t), scriptRelPath)
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("script not found: %s (%v)", scriptPath, err)
	}

	res := s.RunWSM(t, nil, workDir, "runner", scriptPath, "--with-glaze-output", "--output", "json", "--print-result=false")
	if res.ExitCode != 0 {
		t.Fatalf("runner failed for %s:\nstdout:\n%s\nstderr:\n%s", scriptRelPath, res.Stdout, res.Stderr)
	}

	rows := parseRows(t, res.Stdout)
	if len(rows) != 1 {
		t.Fatalf("expected one runner result row for %s, got %d: %s", scriptRelPath, len(rows), res.Stdout)
	}
	row := rows[0]
	if status, _ := row["status"].(string); status != "ok" {
		t.Fatalf("expected runner status=ok for %s, got %#v", scriptRelPath, row)
	}
	if hasResult, _ := row["has_result"].(bool); !hasResult {
		t.Fatalf("expected has_result=true for %s, got %#v", scriptRelPath, row)
	}

	result, ok := row["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object for %s, got %#v", scriptRelPath, row["result"])
	}
	if scriptOK, _ := result["ok"].(bool); !scriptOK {
		t.Fatalf("expected script result ok=true for %s, got %#v", scriptRelPath, result)
	}
	return result
}

func parseRows(t *testing.T, stdout string) []map[string]any {
	t.Helper()

	jsonPart := stdout
	if idx := strings.LastIndex(stdout, "\n["); idx >= 0 {
		jsonPart = stdout[idx+1:]
	} else if idx := strings.Index(stdout, "["); idx >= 0 {
		jsonPart = stdout[idx:]
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(jsonPart), &rows); err != nil {
		t.Fatalf("failed to parse runner JSON output: %v\nstdout:\n%s", err, stdout)
	}
	return rows
}

func asInt(t *testing.T, value any) int {
	t.Helper()

	switch x := value.(type) {
	case float64:
		return int(x)
	case float32:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	default:
		t.Fatalf("expected numeric value, got %T (%#v)", value, value)
		return 0
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}

	dir := wd
	for i := 0; i < 12; i += 1 {
		goModPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(goModPath)
		if err == nil && strings.Contains(string(data), "module github.com/go-go-golems/workspace-manager") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatalf("unable to locate workspace-manager module root from %s", wd)
	return ""
}
