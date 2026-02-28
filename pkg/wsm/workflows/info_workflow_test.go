package workflows

import (
	"testing"
	"time"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
)

func TestInfoWorkflowFieldValue(t *testing.T) {
	wf := NewInfoWorkflow()
	workspace := &wsm.Workspace{
		Name:   "demo",
		Path:   "/tmp/demo",
		Branch: "task/demo",
		Repositories: []wsm.Repository{
			{Name: "repo-a"},
			{Name: "repo-b"},
		},
		Created: time.Date(2026, 2, 28, 10, 30, 45, 0, time.UTC),
	}

	path, err := wf.FieldValue(workspace, "path")
	if err != nil {
		t.Fatalf("unexpected error reading path field: %v", err)
	}
	if path != "/tmp/demo" {
		t.Fatalf("expected /tmp/demo, got %q", path)
	}

	repoCount, err := wf.FieldValue(workspace, "repositories")
	if err != nil {
		t.Fatalf("unexpected error reading repositories field: %v", err)
	}
	if repoCount != "2" {
		t.Fatalf("expected repository count '2', got %q", repoCount)
	}

	if _, err := wf.FieldValue(workspace, "unknown"); err == nil {
		t.Fatalf("expected unknown field to return an error")
	}
}
