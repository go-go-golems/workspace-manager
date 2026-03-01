package registry

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
)

func TestDiscoverResultToRow(t *testing.T) {
	result := &discoverExecutionResult{
		Paths:           []string{"/tmp/a", "/tmp/b"},
		RepositoryCount: 3,
	}

	row := discoverResultToRow(result)
	m := types.RowToMap(row)

	paths, ok := m["paths"].([]string)
	if !ok {
		t.Fatalf("paths field has unexpected type: %T", m["paths"])
	}
	if !reflect.DeepEqual(paths, result.Paths) {
		t.Fatalf("unexpected paths: got %v want %v", paths, result.Paths)
	}

	repoCount, ok := m["repository_count"].(int)
	if !ok {
		t.Fatalf("repository_count field has unexpected type: %T", m["repository_count"])
	}
	if repoCount != result.RepositoryCount {
		t.Fatalf("unexpected repository_count: got %d want %d", repoCount, result.RepositoryCount)
	}
}

func TestReposToRows(t *testing.T) {
	now := time.Date(2026, 3, 1, 10, 15, 0, 0, time.UTC)
	repos := []wsm.Repository{
		{
			Name:          "repo-a",
			Path:          "/tmp/repo-a",
			CurrentBranch: "feature/a",
			RemoteURL:     "git@github.com:go/repo-a.git",
			Categories:    []string{"backend", "go"},
			LastUpdated:   now,
		},
	}

	rows := reposToRows(repos)
	if len(rows) != 1 {
		t.Fatalf("unexpected row count: got %d want 1", len(rows))
	}

	m := types.RowToMap(rows[0])
	if m["name"] != repos[0].Name {
		t.Fatalf("unexpected name: got %v want %s", m["name"], repos[0].Name)
	}
	if m["path"] != repos[0].Path {
		t.Fatalf("unexpected path: got %v want %s", m["path"], repos[0].Path)
	}
	if m["current_branch"] != repos[0].CurrentBranch {
		t.Fatalf("unexpected current_branch: got %v want %s", m["current_branch"], repos[0].CurrentBranch)
	}
	if m["remote_url"] != repos[0].RemoteURL {
		t.Fatalf("unexpected remote_url: got %v want %s", m["remote_url"], repos[0].RemoteURL)
	}

	categories, ok := m["categories"].([]string)
	if !ok {
		t.Fatalf("categories field has unexpected type: %T", m["categories"])
	}
	if !reflect.DeepEqual(categories, repos[0].Categories) {
		t.Fatalf("unexpected categories: got %v want %v", categories, repos[0].Categories)
	}

	lastUpdated, ok := m["last_updated"].(time.Time)
	if !ok {
		t.Fatalf("last_updated field has unexpected type: %T", m["last_updated"])
	}
	if !lastUpdated.Equal(now) {
		t.Fatalf("unexpected last_updated: got %s want %s", lastUpdated, now)
	}
}

func TestWorkspacesToRows(t *testing.T) {
	created := time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC)
	workspaces := []wsm.Workspace{
		{
			Name:       "ws-a",
			Path:       "/tmp/ws-a",
			Branch:     "feature/a",
			BaseBranch: "main",
			Created:    created,
			Repositories: []wsm.Repository{
				{Name: "repo-a"},
				{Name: "repo-b"},
			},
		},
	}

	rows := workspacesToRows(workspaces)
	if len(rows) != 1 {
		t.Fatalf("unexpected row count: got %d want 1", len(rows))
	}

	m := types.RowToMap(rows[0])
	if m["name"] != workspaces[0].Name {
		t.Fatalf("unexpected name: got %v want %s", m["name"], workspaces[0].Name)
	}
	if m["path"] != workspaces[0].Path {
		t.Fatalf("unexpected path: got %v want %s", m["path"], workspaces[0].Path)
	}
	if m["branch"] != workspaces[0].Branch {
		t.Fatalf("unexpected branch: got %v want %s", m["branch"], workspaces[0].Branch)
	}
	if m["base_branch"] != workspaces[0].BaseBranch {
		t.Fatalf("unexpected base_branch: got %v want %s", m["base_branch"], workspaces[0].BaseBranch)
	}

	repoCount, ok := m["repository_count"].(int)
	if !ok {
		t.Fatalf("repository_count field has unexpected type: %T", m["repository_count"])
	}
	if repoCount != 2 {
		t.Fatalf("unexpected repository_count: got %d want 2", repoCount)
	}

	repositories, ok := m["repositories"].([]string)
	if !ok {
		t.Fatalf("repositories field has unexpected type: %T", m["repositories"])
	}
	if !reflect.DeepEqual(repositories, []string{"repo-a", "repo-b"}) {
		t.Fatalf("unexpected repositories: got %v", repositories)
	}

	createdValue, ok := m["created"].(time.Time)
	if !ok {
		t.Fatalf("created field has unexpected type: %T", m["created"])
	}
	if !createdValue.Equal(created) {
		t.Fatalf("unexpected created value: got %s want %s", createdValue, created)
	}
}

func TestPrintReposHuman_NoRepos(t *testing.T) {
	output := captureStdout(t, func() {
		err := printReposHuman(nil, nil)
		if err != nil {
			t.Fatalf("printReposHuman returned error: %v", err)
		}
	})

	if !strings.Contains(output, "No repositories found. Run 'wsm discover' to scan for repositories") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestPrintReposHuman_WithRepos(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	repos := []wsm.Repository{{
		Name:          "repo-a",
		Path:          "/tmp/repo-a",
		CurrentBranch: "main",
		RemoteURL:     "git@github.com:go/repo-a.git",
		Categories:    []string{"backend", "go"},
		LastUpdated:   now,
	}}

	output := captureStdout(t, func() {
		err := printReposHuman(repos, []string{"backend"})
		if err != nil {
			t.Fatalf("printReposHuman returned error: %v", err)
		}
	})

	for _, expected := range []string{
		"Repositories (1)",
		"Filter tags: backend",
		"- repo-a [main]",
		"path: /tmp/repo-a",
		"tags: backend, go",
		"remote: git@github.com:go/repo-a.git",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output missing %q: %q", expected, output)
		}
	}
}

func TestPrintWorkspacesHuman_NoWorkspaces(t *testing.T) {
	output := captureStdout(t, func() {
		err := printWorkspacesHuman(nil)
		if err != nil {
			t.Fatalf("printWorkspacesHuman returned error: %v", err)
		}
	})

	if !strings.Contains(output, "No workspaces found. Use 'wsm create' to create a workspace") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestPrintWorkspacesHuman_WithWorkspaces(t *testing.T) {
	created := time.Date(2026, 3, 1, 13, 45, 0, 0, time.UTC)
	workspaces := []wsm.Workspace{{
		Name:       "ws-a",
		Path:       "/tmp/ws-a",
		Branch:     "feature/a",
		BaseBranch: "main",
		Created:    created,
		Repositories: []wsm.Repository{
			{Name: "repo-a"},
			{Name: "repo-b"},
		},
	}}

	output := captureStdout(t, func() {
		err := printWorkspacesHuman(workspaces)
		if err != nil {
			t.Fatalf("printWorkspacesHuman returned error: %v", err)
		}
	})

	for _, expected := range []string{
		"Workspaces (1)",
		"- ws-a [feature/a]",
		"path: /tmp/ws-a",
		"repos: 2 (repo-a, repo-b)",
		"base: main",
		"created: 2026-03-01 13:45",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output missing %q: %q", expected, output)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	os.Stdout = w
	defer func() {
		os.Stdout = original
	}()

	outputCh := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outputCh <- buf.String()
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer pipe: %v", err)
	}

	return <-outputCh
}
