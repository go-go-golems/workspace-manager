package wsm

import (
	"context"
	"path/filepath"

	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
	"github.com/pkg/errors"
)

// StatusChecker handles workspace status operations
type StatusChecker struct{}

// NewStatusChecker creates a new status checker
func NewStatusChecker() *StatusChecker {
	return &StatusChecker{}
}

// GetWorkspaceStatus gets the status of a workspace
func (sc *StatusChecker) GetWorkspaceStatus(ctx context.Context, workspace *Workspace) (*WorkspaceStatus, error) {
	var repoStatuses []RepositoryStatus

	gc, _ := BuildGitBackends(ctx)
	for _, repo := range workspace.Repositories {
		repoPath := filepath.Join(workspace.Path, repo.Name)
		status, err := sc.getRepositoryStatusWithClient(ctx, repo, repoPath, gc)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get status for repository %s", repo.Name)
		}
		repoStatuses = append(repoStatuses, *status)
	}

	overall := sc.calculateOverallStatus(repoStatuses)

	return &WorkspaceStatus{
		Workspace:    *workspace,
		Repositories: repoStatuses,
		Overall:      overall,
	}, nil
}

// calculateOverallStatus determines the overall workspace status
func (sc *StatusChecker) calculateOverallStatus(repoStatuses []RepositoryStatus) string {
	hasChanges := false
	hasConflicts := false
	needsSync := false

	for _, status := range repoStatuses {
		if status.HasChanges {
			hasChanges = true
		}
		if status.HasConflicts {
			hasConflicts = true
		}
		if status.Ahead > 0 || status.Behind > 0 {
			needsSync = true
		}
	}

	if hasConflicts {
		return "conflicts"
	}
	if hasChanges {
		return "modified"
	}
	if needsSync {
		return "needs-sync"
	}

	return "clean"
}

// getRepositoryStatusWithClient uses the GitClient to compute repository status
func (sc *StatusChecker) getRepositoryStatusWithClient(ctx context.Context, repo Repository, repoPath string, gc gitclient.GitClient) (*RepositoryStatus, error) {
	status := &RepositoryStatus{Repository: repo}

	handle, err := gc.Open(ctx, repoPath)
	if err != nil {
		return nil, errors.Wrap(err, "open repository")
	}

	st, err := gc.Status(ctx, handle)
	if err != nil {
		return nil, errors.Wrap(err, "git status")
	}

	status.CurrentBranch = st.CurrentBranch
	status.ModifiedFiles = st.ModifiedFiles
	status.StagedFiles = st.StagedFiles
	status.UntrackedFiles = st.UntrackedFiles
	status.HasChanges = len(st.ModifiedFiles) > 0 || len(st.StagedFiles) > 0

	if ahead, behind, err := gc.AheadBehind(ctx, handle, ""); err == nil {
		status.Ahead = ahead
		status.Behind = behind
	}

	status.HasConflicts = false
	status.IsMerged = false
	status.NeedsRebase = false

	return status, nil
}
