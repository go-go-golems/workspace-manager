package wsm

import (
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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

	for _, repo := range workspace.Repositories {
		repoPath := filepath.Join(workspace.Path, repo.Name)
		status, err := sc.getRepositoryStatus(ctx, repo, repoPath)
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

// getRepositoryStatus gets the git status of a single repository
func (sc *StatusChecker) getRepositoryStatus(ctx context.Context, repo Repository, repoPath string) (*RepositoryStatus, error) {
	status := &RepositoryStatus{
		Repository: repo,
	}

	// Get current branch
	if branch, err := sc.getCurrentBranch(ctx, repoPath); err == nil {
		status.CurrentBranch = branch
	}

	// Get modified files
	if modifiedFiles, err := sc.getModifiedFiles(ctx, repoPath); err == nil {
		status.ModifiedFiles = modifiedFiles
		status.HasChanges = len(modifiedFiles) > 0
	}

	// Get staged files
	if stagedFiles, err := sc.getStagedFiles(ctx, repoPath); err == nil {
		status.StagedFiles = stagedFiles
		if !status.HasChanges {
			status.HasChanges = len(stagedFiles) > 0
		}
	}

	// Get untracked files
	if untrackedFiles, err := sc.getUntrackedFiles(ctx, repoPath); err == nil {
		status.UntrackedFiles = untrackedFiles
	}

	// Get ahead/behind status
	if ahead, behind, err := sc.getAheadBehind(ctx, repoPath); err == nil {
		status.Ahead = ahead
		status.Behind = behind
	}

	// Check for conflicts
	if hasConflicts, err := sc.hasConflicts(ctx, repoPath); err == nil {
		status.HasConflicts = hasConflicts
	}

	// Check if branch is merged to origin/main
	if isMerged, err := CheckBranchMerged(ctx, repoPath); err == nil {
		status.IsMerged = isMerged
	}

	// Check if branch needs to be rebased on origin/main
	if needsRebase, err := CheckBranchNeedsRebase(ctx, repoPath); err == nil {
		status.NeedsRebase = needsRebase
	}

	return status, nil
}

// getCurrentBranch gets the current branch name
func (sc *StatusChecker) getCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getModifiedFiles gets modified files
func (sc *StatusChecker) getModifiedFiles(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	return files, nil
}

// getStagedFiles gets staged files
func (sc *StatusChecker) getStagedFiles(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--name-only")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	return files, nil
}

// getUntrackedFiles gets untracked files
func (sc *StatusChecker) getUntrackedFiles(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	return files, nil
}

// getAheadBehind gets ahead/behind commit counts
func (sc *StatusChecker) getAheadBehind(ctx context.Context, repoPath string) (int, int, error) {
	// First check if we have a remote tracking branch
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "@{upstream}")
	cmd.Dir = repoPath
	if _, err := cmd.Output(); err != nil {
		// No upstream configured
		return 0, 0, nil
	}

	// Get ahead/behind counts
	cmd = exec.CommandContext(ctx, "git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0, errors.New("unexpected git rev-list output")
	}

	ahead, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	behind, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return ahead, behind, nil
}

// hasConflicts checks if there are merge conflicts
func (sc *StatusChecker) hasConflicts(ctx context.Context, repoPath string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) >= 2 && (line[0] == 'U' || line[1] == 'U' ||
			(line[0] == 'A' && line[1] == 'A') ||
			(line[0] == 'D' && line[1] == 'D')) {
			return true, nil
		}
	}

	return false, nil
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
