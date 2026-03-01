package service

import (
	"context"
	"strings"
	"testing"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
)

func TestManagerNormalizeJobs(t *testing.T) {
	m := NewManager(ManagerOptions{DefaultJobs: 7})

	if got := m.normalizeJobs(3); got != 3 {
		t.Fatalf("expected explicit jobs=3, got %d", got)
	}
	if got := m.normalizeJobs(0); got != 7 {
		t.Fatalf("expected default jobs=7, got %d", got)
	}
	if got := m.normalizeJobs(-1); got != 7 {
		t.Fatalf("expected default jobs=7 for negative input, got %d", got)
	}
}

func TestFilterWorkspaceRepositories(t *testing.T) {
	workspace := &wsm.Workspace{
		Name: "ws-demo",
		Repositories: []wsm.Repository{
			{Name: "repo1"},
			{Name: "repo2"},
		},
	}

	all, err := filterWorkspaceRepositories(workspace, "")
	if err != nil {
		t.Fatalf("unexpected error for empty filter: %v", err)
	}
	if len(all.Repositories) != 2 {
		t.Fatalf("expected all repositories, got %d", len(all.Repositories))
	}

	one, err := filterWorkspaceRepositories(workspace, "repo2")
	if err != nil {
		t.Fatalf("unexpected error filtering repo2: %v", err)
	}
	if len(one.Repositories) != 1 || one.Repositories[0].Name != "repo2" {
		t.Fatalf("unexpected filtered repositories: %#v", one.Repositories)
	}

	_, err = filterWorkspaceRepositories(workspace, "missing")
	if err == nil || !strings.Contains(err.Error(), "repository 'missing' not found") {
		t.Fatalf("expected missing repository error, got: %v", err)
	}
}

func TestManagerLifecycleValidation(t *testing.T) {
	ctx := context.Background()
	m := NewManager(ManagerOptions{})

	_, err := m.LoadWorkspace(ctx, "")
	assertErrorContains(t, err, "workspaceName is required")

	_, err = m.AddRepository(ctx, AddRepositoryInput{})
	assertErrorContains(t, err, "workspaceName is required")

	_, err = m.AddRepository(ctx, AddRepositoryInput{WorkspaceName: "ws"})
	assertErrorContains(t, err, "repoName is required")

	_, err = m.RemoveRepository(ctx, RemoveRepositoryInput{})
	assertErrorContains(t, err, "workspaceName is required")

	_, err = m.RemoveRepository(ctx, RemoveRepositoryInput{WorkspaceName: "ws"})
	assertErrorContains(t, err, "repoName is required")

	_, err = m.DeleteWorkspace(ctx, DeleteWorkspaceInput{})
	assertErrorContains(t, err, "workspaceName is required")

	_, err = m.ForkWorkspace(ctx, ForkWorkspaceInput{})
	assertErrorContains(t, err, "newWorkspaceName is required")

	_, err = m.MergeWorkspace(ctx, MergeWorkspaceInput{})
	assertErrorContains(t, err, "workspaceName is required")
}

func TestManagerGitAndRebaseValidation(t *testing.T) {
	ctx := context.Background()
	m := NewManager(ManagerOptions{})

	_, err := m.BranchCreate(ctx, BranchCreateInput{})
	assertErrorContains(t, err, "branchName is required")

	_, err = m.BranchSwitch(ctx, BranchSwitchInput{})
	assertErrorContains(t, err, "branchName is required")

	missingWorkspace := "__wsmjs_service_missing_workspace__"

	_, err = m.Commit(ctx, CommitInput{
		WorkspaceName: missingWorkspace,
		Message:       "test commit",
	})
	assertError(t, err)

	_, err = m.Diff(ctx, DiffInput{WorkspaceName: missingWorkspace})
	assertError(t, err)

	_, err = m.Log(ctx, LogInput{WorkspaceName: missingWorkspace})
	assertError(t, err)

	_, err = m.BranchList(ctx, BranchListInput{WorkspaceName: missingWorkspace})
	assertError(t, err)

	_, err = m.RebaseRun(ctx, RebaseRunInput{WorkspaceName: missingWorkspace, Manual: true})
	assertError(t, err)

	_, err = m.RebaseStatus(ctx, RebaseStatusInput{WorkspaceName: missingWorkspace})
	assertError(t, err)

	_, err = m.RebaseContinue(ctx, RebaseActionInput{WorkspaceName: missingWorkspace})
	assertError(t, err)

	_, err = m.RebaseAbort(ctx, RebaseActionInput{WorkspaceName: missingWorkspace})
	assertError(t, err)
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected non-nil error")
	}
}
