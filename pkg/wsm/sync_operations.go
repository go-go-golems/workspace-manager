package wsm

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/pkg/errors"
)

// SyncOperations handles synchronization operations across repositories
type SyncOperations struct {
	workspace *Workspace
}

// NewSyncOperations creates a new sync operations handler
func NewSyncOperations(workspace *Workspace) *SyncOperations {
	return &SyncOperations{
		workspace: workspace,
	}
}

// SyncResult represents the result of a sync operation on a repository
type SyncResult struct {
	Repository   string `json:"repository"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
	Pulled       bool   `json:"pulled"`
	Pushed       bool   `json:"pushed"`
	Conflicts    bool   `json:"conflicts"`
	AheadBefore  int    `json:"ahead_before"`
	BehindBefore int    `json:"behind_before"`
	AheadAfter   int    `json:"ahead_after"`
	BehindAfter  int    `json:"behind_after"`
}

// SyncOptions configures sync operations
type SyncOptions struct {
	Pull   bool `json:"pull"`
	Push   bool `json:"push"`
	Rebase bool `json:"rebase"`
	DryRun bool `json:"dry_run"`
}

// SyncWorkspace synchronizes all repositories in the workspace
func (so *SyncOperations) SyncWorkspace(ctx context.Context, options *SyncOptions) ([]SyncResult, error) {
	var results []SyncResult

	output.LogInfo(
		fmt.Sprintf("Starting workspace sync (pull:%v, push:%v, rebase:%v, dry-run:%v)",
			options.Pull, options.Push, options.Rebase, options.DryRun),
		"Starting workspace sync",
		"pull", options.Pull,
		"push", options.Push,
		"rebase", options.Rebase,
		"dry_run", options.DryRun,
	)

	for _, repo := range so.workspace.Repositories {
		repoPath := filepath.Join(so.workspace.Path, repo.Name)
		result := so.syncRepository(ctx, repo.Name, repoPath, options)
		results = append(results, result)
	}

	return results, nil
}

// syncRepository synchronizes a single repository
func (so *SyncOperations) syncRepository(ctx context.Context, repoName, repoPath string, options *SyncOptions) SyncResult {
	result := SyncResult{
		Repository: repoName,
		Success:    true,
	}

	// Get initial ahead/behind status
	ahead, behind, err := so.getAheadBehind(ctx, repoPath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to get initial status: %v", err)
		return result
	}
	result.AheadBefore = ahead
	result.BehindBefore = behind

	if options.DryRun {
		result.Error = "dry-run mode"
		return result
	}

	// Pull changes if requested
	if options.Pull {
		if err := so.pullRepository(ctx, repoPath, options.Rebase); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("pull failed: %v", err)
			result.Conflicts = so.hasConflicts(ctx, repoPath)
			return result
		}
		result.Pulled = true
	}

	// Push changes if requested
	if options.Push {
		if err := so.pushRepository(ctx, repoPath); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("push failed: %v", err)
			return result
		}
		result.Pushed = true
	}

	// Get final ahead/behind status
	ahead, behind, err = so.getAheadBehind(ctx, repoPath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to get final status: %v", err)
		return result
	}
	result.AheadAfter = ahead
	result.BehindAfter = behind

	output.LogInfo(
		fmt.Sprintf("Synced %s (ahead: %d→%d, behind: %d→%d)", repoName, result.AheadBefore, result.AheadAfter, result.BehindBefore, result.BehindAfter),
		"Repository sync completed",
		"repository", repoName,
		"pulled", result.Pulled,
		"pushed", result.Pushed,
		"ahead_before", result.AheadBefore,
		"behind_before", result.BehindBefore,
		"ahead_after", result.AheadAfter,
		"behind_after", result.BehindAfter,
	)

	return result
}

// pullRepository pulls changes from remote
func (so *SyncOperations) pullRepository(ctx context.Context, repoPath string, rebase bool) error {
	var cmd *exec.Cmd
	if rebase {
		cmd = exec.CommandContext(ctx, "git", "pull", "--rebase")
	} else {
		cmd = exec.CommandContext(ctx, "git", "pull")
	}
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "git pull failed: %s", string(output))
	}

	return nil
}

// pushRepository pushes changes to remote
func (so *SyncOperations) pushRepository(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "push")
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "git push failed: %s", string(output))
	}

	return nil
}

// getAheadBehind gets ahead/behind counts
func (so *SyncOperations) getAheadBehind(ctx context.Context, repoPath string) (int, int, error) {
	// Check if we have a remote tracking branch
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

	var ahead, behind int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(output)), "%d\t%d", &ahead, &behind); err != nil {
		return 0, 0, err
	}

	return ahead, behind, nil
}

// hasConflicts checks if there are merge conflicts
func (so *SyncOperations) hasConflicts(ctx context.Context, repoPath string) bool {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) >= 2 && (line[0] == 'U' || line[1] == 'U' ||
			(line[0] == 'A' && line[1] == 'A') ||
			(line[0] == 'D' && line[1] == 'D')) {
			return true
		}
	}

	return false
}

// CreateBranch creates a branch across all repositories
func (so *SyncOperations) CreateBranch(ctx context.Context, branchName string, track bool) ([]SyncResult, error) {
	var results []SyncResult

	output.LogInfo(
		fmt.Sprintf("Creating branch '%s' across workspace", branchName),
		"Creating branch across workspace",
		"branch", branchName,
		"track", track,
	)

	for _, repo := range so.workspace.Repositories {
		repoPath := filepath.Join(so.workspace.Path, repo.Name)
		result := so.createBranchInRepository(ctx, repo.Name, repoPath, branchName, track)
		results = append(results, result)
	}

	return results, nil
}

// createBranchInRepository creates a branch in a single repository
func (so *SyncOperations) createBranchInRepository(ctx context.Context, repoName, repoPath, branchName string, track bool) SyncResult {
	result := SyncResult{
		Repository: repoName,
		Success:    true,
	}

	var cmd *exec.Cmd
	if track {
		cmd = exec.CommandContext(ctx, "git", "checkout", "-b", branchName, "--track")
	} else {
		cmd = exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
	}
	cmd.Dir = repoPath

	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create branch: %s", string(cmdOutput))
		return result
	}

	output.LogInfo(
		fmt.Sprintf("Created branch '%s' in %s", branchName, repoName),
		"Branch created successfully",
		"repository", repoName,
		"branch", branchName,
	)

	return result
}

// SwitchBranch switches all repositories to a specific branch
func (so *SyncOperations) SwitchBranch(ctx context.Context, branchName string) ([]SyncResult, error) {
	var results []SyncResult

	output.LogInfo(
		fmt.Sprintf("Switching to branch '%s' across workspace", branchName),
		"Switching branch across workspace",
		"branch", branchName,
	)

	for _, repo := range so.workspace.Repositories {
		repoPath := filepath.Join(so.workspace.Path, repo.Name)
		result := so.switchBranchInRepository(ctx, repo.Name, repoPath, branchName)
		results = append(results, result)
	}

	return results, nil
}

// switchBranchInRepository switches branch in a single repository
func (so *SyncOperations) switchBranchInRepository(ctx context.Context, repoName, repoPath, branchName string) SyncResult {
	result := SyncResult{
		Repository: repoName,
		Success:    true,
	}

	cmd := exec.CommandContext(ctx, "git", "checkout", branchName)
	cmd.Dir = repoPath

	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to switch branch: %s", string(cmdOutput))
		return result
	}

	output.LogInfo(
		fmt.Sprintf("Switched to branch '%s' in %s", branchName, repoName),
		"Branch switched successfully",
		"repository", repoName,
		"branch", branchName,
	)

	return result
}

// GetWorkspaceLog gets commit history across workspace
func (so *SyncOperations) GetWorkspaceLog(ctx context.Context, since string, oneline bool, limit int) (map[string]string, error) {
	logs := make(map[string]string)

	for _, repo := range so.workspace.Repositories {
		repoPath := filepath.Join(so.workspace.Path, repo.Name)
		log, err := so.getRepositoryLog(ctx, repoPath, since, oneline, limit)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get log for %s", repo.Name)
		}
		if log != "" {
			logs[repo.Name] = log
		}
	}

	return logs, nil
}

// getRepositoryLog gets commit history for a single repository
func (so *SyncOperations) getRepositoryLog(ctx context.Context, repoPath, since string, oneline bool, limit int) (string, error) {
	args := []string{"log"}

	if since != "" {
		args = append(args, "--since", since)
	}

	if oneline {
		args = append(args, "--oneline")
	}

	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
