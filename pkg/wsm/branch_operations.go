package wsm

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	branchsvc "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"github.com/pkg/errors"
)

// BranchOperationResult represents the result of a branch operation on a repository.
type BranchOperationResult struct {
	Repository string `json:"repository"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// BranchOperations handles branch operations across repositories.
type BranchOperations struct {
	workspace *Workspace
}

// NewBranchOperations creates a branch operations service.
func NewBranchOperations(workspace *Workspace) *BranchOperations {
	return &BranchOperations{workspace: workspace}
}

// CreateBranch creates a branch across all repositories in the workspace.
func (bo *BranchOperations) CreateBranch(ctx context.Context, branchName string, track bool) ([]BranchOperationResult, error) {
	var results []BranchOperationResult

	output.LogInfo(
		fmt.Sprintf("Creating branch '%s' across workspace", branchName),
		"Creating branch across workspace",
		"branch", branchName,
		"track", track,
	)

	for _, repo := range bo.workspace.Repositories {
		repoPath := filepath.Join(bo.workspace.Path, repo.Name)
		result := bo.createBranchInRepository(ctx, repo.Name, repoPath, branchName, track)
		results = append(results, result)
	}

	return results, nil
}

func (bo *BranchOperations) createBranchInRepository(ctx context.Context, repoName, repoPath, branchName string, track bool) BranchOperationResult {
	result := BranchOperationResult{
		Repository: repoName,
		Success:    true,
	}

	gc, _ := BuildGitBackends(ctx)
	h, err := gc.Open(ctx, repoPath)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}
	if err := gc.CreateBranch(ctx, h, branchName, track, ""); err != nil {
		result.Success = false
		result.Error = err.Error()
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

// SwitchBranch switches all repositories to a branch in the workspace.
func (bo *BranchOperations) SwitchBranch(ctx context.Context, branchName string) ([]BranchOperationResult, error) {
	var results []BranchOperationResult

	output.LogInfo(
		fmt.Sprintf("Switching to branch '%s' across workspace", branchName),
		"Switching branch across workspace",
		"branch", branchName,
	)

	for _, repo := range bo.workspace.Repositories {
		repoPath := filepath.Join(bo.workspace.Path, repo.Name)
		result := bo.switchBranchInRepository(ctx, repo.Name, repoPath, branchName)
		results = append(results, result)
	}

	return results, nil
}

func (bo *BranchOperations) switchBranchInRepository(ctx context.Context, repoName, repoPath, branchName string) BranchOperationResult {
	result := BranchOperationResult{
		Repository: repoName,
		Success:    true,
	}

	gc, _ := BuildGitBackends(ctx)
	branches := BuildBranchService(ctx)
	h, err := gc.Open(ctx, repoPath)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	plan, err := branches.Resolve(ctx, repoPath, branchsvc.BranchResolutionRequest{
		TargetBranch: branchsvc.BranchName(branchName),
		Remote:       branchsvc.DefaultRemoteName,
		Mode:         branchsvc.ResolutionModeSync,
	})
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	switch plan.Strategy {
	case branchsvc.ResolutionStrategyUseLocal:
		err = gc.CheckoutBranch(ctx, h, branchName, false, false)
	case branchsvc.ResolutionStrategyTrackRemote:
		err = gc.CreateBranch(ctx, h, branchName, true, plan.RemoteRef)
	case branchsvc.ResolutionStrategyCreateFromBase:
		err = gc.CreateBranch(ctx, h, branchName, false, plan.StartPoint)
	case branchsvc.ResolutionStrategyCreateFromHead:
		err = gc.CreateBranch(ctx, h, branchName, false, "")
	case branchsvc.ResolutionStrategyUnspecified:
		err = errors.New("resolution strategy unspecified")
	}
	if err != nil {
		result.Success = false
		result.Error = err.Error()
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
