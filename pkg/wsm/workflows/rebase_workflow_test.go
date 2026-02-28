package workflows

import (
	"testing"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
)

func TestRebaseWorkflowManualPlan_AllRepos(t *testing.T) {
	workspace := &wsm.Workspace{
		Path: "/tmp/ws",
		Repositories: []wsm.Repository{
			{Name: "repo-a"},
			{Name: "repo-b"},
		},
	}
	wf := NewRebaseWorkflow(workspace)

	plan := wf.ManualPlan("", "main")
	if len(plan) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(plan))
	}
}

func TestRebaseWorkflowManualPlan_SingleRepo(t *testing.T) {
	workspace := &wsm.Workspace{
		Path: "/tmp/ws",
		Repositories: []wsm.Repository{
			{Name: "repo-a"},
			{Name: "repo-b"},
		},
	}
	wf := NewRebaseWorkflow(workspace)

	plan := wf.ManualPlan("repo-b", "develop")
	if len(plan) != 1 {
		t.Fatalf("expected 1 command, got %d", len(plan))
	}
	if plan[0] == "" {
		t.Fatalf("expected non-empty command")
	}
}
