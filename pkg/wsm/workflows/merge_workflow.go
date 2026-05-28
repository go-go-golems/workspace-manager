package workflows

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	branchsvc "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"github.com/pkg/errors"
)

// MergeRequest captures merge execution options.
type MergeRequest struct {
	WorkspaceName string
	DryRun        bool
	Force         bool
	KeepWorkspace bool
}

// MergeCandidate describes repository-specific merge plan state.
type MergeCandidate struct {
	Repository    wsm.Repository
	WorktreePath  string
	BaseBranch    string
	CurrentBranch string
	HasChanges    bool
	IsClean       bool
}

// MergeWorkflow orchestrates workspace merge behavior.
type MergeWorkflow struct{}

// NewMergeWorkflow creates a merge workflow service.
func NewMergeWorkflow() *MergeWorkflow {
	return &MergeWorkflow{}
}

// Execute runs the merge workflow.
func (mw *MergeWorkflow) Execute(ctx context.Context, req MergeRequest) error {
	workspaceName := req.WorkspaceName
	workspaceResolver := wsm.NewWorkspaceContextService()

	if workspaceName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to get current directory")
		}
		detected, err := workspaceResolver.DetectWorkspaceName(cwd)
		if err != nil {
			return errors.Wrap(err, "failed to detect workspace. Use 'workspace-manager merge <workspace-name>' or specify --workspace flag")
		}
		workspaceName = detected
	}

	workspace, err := workspaceResolver.LoadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	if workspace.BaseBranch == "" {
		return errors.New("workspace is not a fork (no base branch specified). Only forked workspaces can be merged.")
	}

	baseWorkspace, err := mw.findWorkspaceByBranch(workspace.BaseBranch)
	if err != nil {
		return errors.Wrap(err, "failed to check for base branch workspace")
	}
	if baseWorkspace != nil {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to get current directory")
		}
		if !strings.HasPrefix(cwd, baseWorkspace.Path) {
			return errors.Errorf("found workspace '%s' for base branch '%s'. Please run the merge command from within that workspace (at %s) to avoid git worktree conflicts",
				baseWorkspace.Name, workspace.BaseBranch, baseWorkspace.Path)
		}
		output.PrintInfo("✓ Running merge from base workspace '%s' as required", baseWorkspace.Name)
	}

	output.PrintInfo("Merging workspace '%s' (branch: %s → %s)", workspace.Name, workspace.Branch, workspace.BaseBranch)

	checker := wsm.NewStatusChecker()
	status, err := checker.GetWorkspaceStatus(ctx, workspace)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace status")
	}

	candidates := make([]MergeCandidate, 0, len(status.Repositories))
	uncleanRepos := make([]string, 0)
	for _, repoStatus := range status.Repositories {
		candidate := MergeCandidate{
			Repository:    repoStatus.Repository,
			WorktreePath:  filepath.Join(workspace.Path, repoStatus.Repository.Name),
			BaseBranch:    workspace.BaseBranch,
			CurrentBranch: repoStatus.CurrentBranch,
			HasChanges:    repoStatus.HasChanges,
			IsClean:       !repoStatus.HasChanges && len(repoStatus.StagedFiles) == 0 && len(repoStatus.UntrackedFiles) == 0,
		}
		candidates = append(candidates, candidate)
		if !candidate.IsClean {
			uncleanRepos = append(uncleanRepos, repoStatus.Repository.Name)
		}
	}

	if len(uncleanRepos) > 0 && !req.Force {
		return errors.Errorf("the following repositories have uncommitted changes: %s. Commit or stash changes first, or use --force", strings.Join(uncleanRepos, ", "))
	}

	for _, candidate := range candidates {
		if candidate.CurrentBranch != workspace.Branch {
			return errors.Errorf("repository '%s' is on branch '%s', expected '%s'. Switch all repositories to the workspace branch first",
				candidate.Repository.Name, candidate.CurrentBranch, workspace.Branch)
		}
	}

	if req.DryRun {
		return mw.previewMerge(workspace, candidates)
	}

	if !req.Force {
		confirmed, err := mw.confirmMerge(workspace, candidates, req.KeepWorkspace)
		if err != nil {
			return errors.Wrap(err, "failed to get user confirmation")
		}
		if !confirmed {
			output.PrintInfo("Merge cancelled by user")
			return nil
		}
	}

	return mw.executeMerge(ctx, workspace, candidates, req.KeepWorkspace)
}

func (mw *MergeWorkflow) previewMerge(workspace *wsm.Workspace, candidates []MergeCandidate) error {
	output.PrintHeader("📋 Merge Preview: %s", workspace.Name)
	fmt.Println()

	output.PrintInfo("Workspace Details:")
	fmt.Printf("  Name: %s\n", workspace.Name)
	fmt.Printf("  Path: %s\n", workspace.Path)
	fmt.Printf("  Current branch: %s\n", workspace.Branch)
	fmt.Printf("  Base branch: %s\n", workspace.BaseBranch)
	fmt.Println()

	output.PrintInfo("Merge Plan:")
	for _, candidate := range candidates {
		status := "✓ Clean"
		if !candidate.IsClean {
			status = "⚠️  Has changes"
		}

		fmt.Printf("  %s (%s)\n", candidate.Repository.Name, status)
		fmt.Printf("    Merge: %s → %s\n", workspace.Branch, workspace.BaseBranch)
		fmt.Printf("    Push: %s to origin\n", workspace.BaseBranch)
	}

	fmt.Println()
	output.PrintInfo("After successful merge:")
	fmt.Printf("  - All repositories will have %s branch updated\n", workspace.BaseBranch)
	fmt.Printf("  - Changes will be pushed to origin\n")
	fmt.Printf("  - Workspace will be deleted\n")

	return nil
}

func (mw *MergeWorkflow) confirmMerge(workspace *wsm.Workspace, candidates []MergeCandidate, keepWorkspace bool) (bool, error) {
	fmt.Printf("\n")
	output.PrintWarning("You are about to merge workspace '%s'", workspace.Name)
	fmt.Printf("  Branch: %s → %s\n", workspace.Branch, workspace.BaseBranch)
	fmt.Printf("  Repositories: %d\n", len(candidates))

	if !keepWorkspace {
		fmt.Printf("  The workspace will be DELETED after successful merge\n")
	}
	fmt.Println()

	hasChanges := false
	for _, candidate := range candidates {
		if !candidate.IsClean {
			if !hasChanges {
				output.PrintWarning("The following repositories have uncommitted changes:")
				hasChanges = true
			}
			fmt.Printf("  - %s\n", candidate.Repository.Name)
		}
	}
	if hasChanges {
		fmt.Printf("\nThese changes will be included in the merge.\n")
	}

	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Do you want to proceed with the merge?").
				Value(&confirmed),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}

	return confirmed, nil
}

func (mw *MergeWorkflow) executeMerge(ctx context.Context, workspace *wsm.Workspace, candidates []MergeCandidate, keepWorkspace bool) error {
	output.PrintHeader("🔀 Executing Merge: %s", workspace.Name)

	successfulMerges := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		output.PrintInfo("Processing repository: %s", candidate.Repository.Name)

		if err := mw.mergeRepository(ctx, candidate); err != nil {
			output.PrintError("Failed to merge repository %s: %v", candidate.Repository.Name, err)

			if len(successfulMerges) > 0 {
				output.PrintWarning("Rolling back successful merges due to failure...")
				mw.rollbackMerges(ctx, workspace, successfulMerges)
			}

			return errors.Wrapf(err, "merge failed for repository %s", candidate.Repository.Name)
		}

		successfulMerges = append(successfulMerges, candidate.Repository.Name)
		output.PrintSuccess("✓ Successfully merged %s", candidate.Repository.Name)
	}

	output.PrintSuccess("All repositories merged successfully!")

	if !keepWorkspace {
		output.PrintInfo("Deleting workspace '%s'...", workspace.Name)

		wm, err := wsm.NewWorkspaceManager()
		if err != nil {
			return errors.Wrap(err, "failed to create workspace manager for deletion")
		}

		if err := wm.DeleteWorkspace(ctx, workspace.Name, true, true); err != nil {
			output.PrintWarning("Failed to delete workspace: %v", err)
			output.PrintInfo("You may need to delete it manually: workspace-manager delete %s", workspace.Name)
		} else {
			output.PrintSuccess("✓ Workspace '%s' deleted successfully", workspace.Name)
		}
	}

	fmt.Println()
	output.PrintSuccess("Merge completed successfully!")
	output.PrintInfo("Summary:")
	fmt.Printf("  - Merged %d repositories\n", len(successfulMerges))
	fmt.Printf("  - Branch %s merged into %s\n", workspace.Branch, workspace.BaseBranch)
	fmt.Printf("  - Changes pushed to origin\n")
	if !keepWorkspace {
		fmt.Printf("  - Workspace deleted\n")
	}

	return nil
}

func (mw *MergeWorkflow) mergeRepository(ctx context.Context, candidate MergeCandidate) error {
	repoPath := candidate.WorktreePath

	log.Debug().
		Str("repository", candidate.Repository.Name).
		Str("repoPath", repoPath).
		Str("currentBranch", candidate.CurrentBranch).
		Str("baseBranch", candidate.BaseBranch).
		Msg("Starting repository merge")

	output.PrintInfo("  Fetching latest changes...")
	if err := executeGitCommand(ctx, repoPath, "git", "fetch", "origin"); err != nil {
		return errors.Wrap(err, "failed to fetch latest changes")
	}

	output.PrintInfo("  Switching to base branch: %s", candidate.BaseBranch)
	if err := executeGitCommand(ctx, repoPath, "git", "checkout", candidate.BaseBranch); err != nil {
		return errors.Wrapf(err, "failed to switch to base branch %s", candidate.BaseBranch)
	}

	output.PrintInfo("  Pulling latest base branch changes...")
	if err := executeGitCommand(ctx, repoPath, "git", "pull", "origin", candidate.BaseBranch); err != nil {
		return errors.Wrapf(err, "failed to pull latest changes for %s", candidate.BaseBranch)
	}

	output.PrintInfo("  Merging %s into %s...", candidate.CurrentBranch, candidate.BaseBranch)
	if err := executeGitCommand(ctx, repoPath, "git", "merge", candidate.CurrentBranch); err != nil {
		if isGitMergeConflict(err) {
			return errors.Errorf("merge conflict detected in %s. Please resolve conflicts manually and retry", candidate.Repository.Name)
		}
		return errors.Wrapf(err, "failed to merge %s into %s", candidate.CurrentBranch, candidate.BaseBranch)
	}

	output.PrintInfo("  Pushing merged changes...")
	if err := executeGitCommand(ctx, repoPath, "git", "push", "origin", candidate.BaseBranch); err != nil {
		return errors.Wrapf(err, "failed to push merged changes for %s", candidate.BaseBranch)
	}

	log.Debug().
		Str("repository", candidate.Repository.Name).
		Msg("Repository merge completed successfully")

	return nil
}

func executeGitCommand(ctx context.Context, repoPath string, args ...string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = repoPath

	cmdStr := strings.Join(args, " ")
	log.Debug().Str("command", cmdStr).Str("repoPath", repoPath).Msg("Executing git command")

	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug().
			Str("command", cmdStr).
			Str("repoPath", repoPath).
			Str("output", string(cmdOutput)).
			Err(err).
			Msg("Git command failed")

		return errors.Wrapf(err, "git command failed: %s (output: %s)", cmdStr, string(cmdOutput))
	}

	log.Debug().
		Str("command", cmdStr).
		Str("repoPath", repoPath).
		Str("output", string(cmdOutput)).
		Msg("Git command succeeded")

	return nil
}

func isGitMergeConflict(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "conflict") ||
		strings.Contains(errStr, "merge failed") ||
		strings.Contains(errStr, "automatic merge failed")
}

func (mw *MergeWorkflow) rollbackMerges(ctx context.Context, workspace *wsm.Workspace, successfulMerges []string) {
	output.PrintWarning("🔄 Rolling back %d successful merges...", len(successfulMerges))

	for _, repoName := range successfulMerges {
		repoPath := filepath.Join(workspace.Path, repoName)
		output.PrintInfo("  Rolling back %s...", repoName)

		if err := executeGitCommand(ctx, repoPath, "git", "checkout", workspace.BaseBranch); err != nil {
			output.PrintWarning("    Failed to checkout %s: %v", workspace.BaseBranch, err)
			continue
		}

		remoteBaseRef := branchsvc.RemoteTrackingRef(branchsvc.DefaultRemoteName, branchsvc.BranchName(workspace.BaseBranch))
		if err := executeGitCommand(ctx, repoPath, "git", "reset", "--hard", remoteBaseRef); err != nil {
			output.PrintWarning("    Failed to reset %s: %v", workspace.BaseBranch, err)
			continue
		}

		if err := executeGitCommand(ctx, repoPath, "git", "checkout", workspace.Branch); err != nil {
			output.PrintWarning("    Failed to checkout %s: %v", workspace.Branch, err)
		}

		output.PrintInfo("    ✓ Rolled back %s", repoName)
	}

	output.PrintInfo("🔄 Rollback completed")
}

func (mw *MergeWorkflow) findWorkspaceByBranch(branchName string) (*wsm.Workspace, error) {
	workspaces, err := wsm.LoadWorkspaces()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load workspaces")
	}

	for _, workspace := range workspaces {
		if workspace.Branch == branchName {
			return &workspace, nil
		}
	}

	return nil, nil
}
