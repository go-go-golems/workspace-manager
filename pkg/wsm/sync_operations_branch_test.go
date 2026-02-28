package wsm

import (
	"context"
	"testing"
)

func TestSyncSwitchBranch_RemoteTrackingBranch_BackendMatrix(t *testing.T) {
	ctx := context.Background()

	for _, backend := range []string{"cli", "gogit", "hybrid"} {
		t.Run(backend, func(t *testing.T) {
			t.Setenv("WSM_GIT_BACKEND", backend)
			repoPath := createWorkspaceBranchFixture(t)
			so := &SyncOperations{workspace: &Workspace{}}

			result := so.switchBranchInRepository(ctx, "repo", repoPath, "feature/remote-only")
			if !result.Success {
				t.Fatalf("expected success, got error: %s", result.Error)
			}

			current := runGitInDirOrFail(t, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
			if current != "feature/remote-only" {
				t.Fatalf("expected current branch feature/remote-only, got %s", current)
			}
		})
	}
}

func TestSyncSwitchBranch_MissingBranchCreatesFromHead(t *testing.T) {
	ctx := context.Background()
	repoPath := createWorkspaceBranchFixture(t)
	so := &SyncOperations{workspace: &Workspace{}}

	result := so.switchBranchInRepository(ctx, "repo", repoPath, "feature/new-local")
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	current := runGitInDirOrFail(t, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if current != "feature/new-local" {
		t.Fatalf("expected current branch feature/new-local, got %s", current)
	}
}
