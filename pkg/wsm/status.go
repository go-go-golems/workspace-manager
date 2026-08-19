package wsm

import (
	"context"
	"path/filepath"

	branchsvc "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// StatusChecker handles workspace status operations
type StatusChecker struct{}

// NewStatusChecker creates a new status checker
func NewStatusChecker() *StatusChecker {
	return &StatusChecker{}
}

// StatusOptions configures workspace status collection.
type StatusOptions struct {
	MaxJobs int
	Fetch   bool
}

// GetWorkspaceStatus gets the status of a workspace
func (sc *StatusChecker) GetWorkspaceStatus(ctx context.Context, workspace *Workspace) (*WorkspaceStatus, error) {
	return sc.GetWorkspaceStatusWithOptions(ctx, workspace, StatusOptions{MaxJobs: 1})
}

// GetWorkspaceStatusWithOptions gets the status of a workspace with options (e.g., concurrency)
func (sc *StatusChecker) GetWorkspaceStatusWithOptions(ctx context.Context, workspace *Workspace, opts StatusOptions) (*WorkspaceStatus, error) {
	maxJobs := opts.MaxJobs
	if maxJobs < 1 {
		maxJobs = 1
	}

	repoCount := len(workspace.Repositories)
	repoStatuses := make([]RepositoryStatus, repoCount)

	gc, _ := BuildGitBackends(ctx)

	if maxJobs == 1 || repoCount <= 1 {
		for i, repo := range workspace.Repositories {
			repoPath := filepath.Join(workspace.Path, repo.Name)
			status, err := sc.getRepositoryStatusWithClient(ctx, repo, repoPath, workspace.BaseBranch, opts.Fetch, gc)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get status for repository %s", repo.Name)
			}
			repoStatuses[i] = *status
		}
	} else {
		sem := semaphore.NewWeighted(int64(maxJobs))
		g, gctx := errgroup.WithContext(ctx)

		for i := range workspace.Repositories {
			i := i
			if err := sem.Acquire(gctx, 1); err != nil {
				return nil, err
			}
			g.Go(func() error {
				defer sem.Release(1)
				repo := workspace.Repositories[i]
				repoPath := filepath.Join(workspace.Path, repo.Name)
				status, err := sc.getRepositoryStatusWithClient(gctx, repo, repoPath, workspace.BaseBranch, opts.Fetch, gc)
				if err != nil {
					return errors.Wrapf(err, "failed to get status for repository %s", repo.Name)
				}
				repoStatuses[i] = *status
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
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
func (sc *StatusChecker) getRepositoryStatusWithClient(ctx context.Context, repo Repository, repoPath string, baseBranch string, fetch bool, gc gitclient.GitClient) (*RepositoryStatus, error) {
	status := &RepositoryStatus{Repository: repo}

	handle, err := gc.Open(ctx, repoPath)
	if err != nil {
		return nil, errors.Wrap(err, "open repository")
	}
	if fetch {
		if err := gc.Fetch(ctx, handle, ""); err != nil {
			return nil, errors.Wrap(err, "git fetch origin")
		}
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

	// Resolve the effective base branch + remote for this repo via the full
	// precedence (in-workspace override > config-dir override > workspace base
	// > discovered default > env > main). The repo's BaseBranch/BaseRemote fields
	// carry the per-repo override (overlaid from .wsm/wsm.json at load time, see
	// LoadWorkspace); baseBranch is the workspace-level base. Centralizing this
	// here means the checks cannot forget a layer (e.g. the empty->main fallback
	// that bit us in E1 part 2).
	base, remote := branchsvc.ResolveBaseBranchForRepo(branchsvc.RepoBaseInput{
		BaseBranchWorkspace: repo.BaseBranchWorkspace,
		BaseRemoteWorkspace: repo.BaseRemoteWorkspace,
		BaseBranchGlobal:    repo.BaseBranch,
		BaseRemoteGlobal:    repo.BaseRemote,
		WorkspaceBase:       baseBranch,
		DefaultBaseBranch:   repo.DefaultBaseBranch,
	})

	// Compute merge/rebase status against the resolved base ref (prefer
	// remote-tracking, fall back to local, else unknown). The result carries
	// provenance (which ref, why if it could not compare); the bool mirrors are
	// kept for JSON compatibility with existing consumers.
	mergedCmp, _ := CheckBranchMerged(ctx, gc, repoPath, string(base), string(remote))
	status.Base = mergedCmp
	status.IsMerged = mergedCmp.IsMerged

	// Always honor the rebase comparison's outcome, even if it returned an
	// error: a failed comparison must surface as BaseError (e.g. context
	// expiry or a git object error), not be swallowed into a confident
	// NeedsRebase=false. If the rebase check errored where the merge check
	// resolved, promote the error so the status table shows '!' rather than a
	// misleading rebase checkmark.
	rebaseCmp, _ := CheckBranchNeedsRebase(ctx, gc, repoPath, string(base), string(remote))
	status.Base.NeedsRebase = rebaseCmp.NeedsRebase
	status.NeedsRebase = rebaseCmp.NeedsRebase
	if rebaseCmp.Status == BaseError && mergedCmp.Status != BaseError {
		status.Base.Status = rebaseCmp.Status
		status.Base.Reason = rebaseCmp.Reason
	}

	return status, nil
}
