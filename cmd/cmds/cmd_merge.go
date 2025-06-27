package cmds

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
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewMergeCommand() *cobra.Command {
	var (
		dryRun        bool
		force         bool
		workspace     string
		keepWorkspace bool
	)

	cmd := &cobra.Command{
		Use:   "merge [workspace-name]",
		Short: "Merge a forked workspace back into its base branch and delete the workspace",
		Long: `Merge a forked workspace back into its base branch and optionally delete the workspace.

This command:
1. Detects the current workspace (if not specified)
2. Verifies the workspace is a fork (has a base branch)
3. Checks that all repositories are clean before merging
4. For each repository:
   - Switches to the base branch
   - Merges the workspace branch into the base branch
   - Pushes the merged changes
5. Optionally deletes the workspace after successful merge

The command handles merge conflicts gracefully and provides rollback on failure.

Examples:
  # Merge current workspace back to its base branch
  workspace-manager merge

  # Merge a specific workspace
  workspace-manager merge my-feature-workspace

  # Preview what would be merged without executing
  workspace-manager merge --dry-run

  # Merge without asking for confirmation
  workspace-manager merge --force

  # Merge but keep the workspace (don't delete)
  workspace-manager merge --keep-workspace`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := workspace
			if len(args) > 0 {
				workspaceName = args[0]
			}
			return runMerge(cmd.Context(), workspaceName, dryRun, force, keepWorkspace)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be merged without executing")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	cmd.Flags().BoolVar(&keepWorkspace, "keep-workspace", false, "Keep the workspace after merge (don't delete it)")

	return cmd
}

type MergeCandidate struct {
	Repository    wsm.Repository
	WorktreePath  string
	BaseBranch    string
	CurrentBranch string
	HasChanges    bool
	IsClean       bool
}

func runMerge(ctx context.Context, workspaceName string, dryRun, force, keepWorkspace bool) error {
	// Detect workspace if not specified
	if workspaceName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to get current directory")
		}

		detected, err := detectWorkspace(cwd)
		if err != nil {
			return errors.Wrap(err, "failed to detect workspace. Use 'workspace-manager merge <workspace-name>' or specify --workspace flag")
		}
		workspaceName = detected
	}

	// Load workspace
	workspace, err := loadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	// Verify this is a forked workspace
	if workspace.BaseBranch == "" {
		return errors.New("workspace is not a fork (no base branch specified). Only forked workspaces can be merged.")
	}

	output.PrintInfo("Merging workspace '%s' (branch: %s → %s)", workspace.Name, workspace.Branch, workspace.BaseBranch)

	// Get workspace status to verify readiness for merge
	checker := wsm.NewStatusChecker()
	status, err := checker.GetWorkspaceStatus(ctx, workspace)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace status")
	}

	// Prepare merge candidates
	var candidates []MergeCandidate
	var uncleanRepos []string

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

	// Check for unclean repositories
	if len(uncleanRepos) > 0 && !force {
		return errors.Errorf("the following repositories have uncommitted changes: %s. Commit or stash changes first, or use --force", strings.Join(uncleanRepos, ", "))
	}

	// Verify all repositories are on the workspace branch
	for _, candidate := range candidates {
		if candidate.CurrentBranch != workspace.Branch {
			return errors.Errorf("repository '%s' is on branch '%s', expected '%s'. Switch all repositories to the workspace branch first",
				candidate.Repository.Name, candidate.CurrentBranch, workspace.Branch)
		}
	}

	if dryRun {
		return previewMerge(workspace, candidates)
	}

	// Ask for confirmation unless force is set
	if !force {
		confirmed, err := confirmMerge(workspace, candidates, keepWorkspace)
		if err != nil {
			return errors.Wrap(err, "failed to get user confirmation")
		}
		if !confirmed {
			output.PrintInfo("Merge cancelled by user")
			return nil
		}
	}

	// Execute merge
	return executeMerge(ctx, workspace, candidates, keepWorkspace)
}

func previewMerge(workspace *wsm.Workspace, candidates []MergeCandidate) error {
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

func confirmMerge(workspace *wsm.Workspace, candidates []MergeCandidate, keepWorkspace bool) (bool, error) {
	fmt.Printf("\n")
	output.PrintWarning("You are about to merge workspace '%s'", workspace.Name)
	fmt.Printf("  Branch: %s → %s\n", workspace.Branch, workspace.BaseBranch)
	fmt.Printf("  Repositories: %d\n", len(candidates))

	if !keepWorkspace {
		fmt.Printf("  The workspace will be DELETED after successful merge\n")
	}
	fmt.Println()

	// Show any repositories with changes
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

	err := form.Run()
	if err != nil {
		return false, err
	}

	return confirmed, nil
}

func executeMerge(ctx context.Context, workspace *wsm.Workspace, candidates []MergeCandidate, keepWorkspace bool) error {
	output.PrintHeader("🔀 Executing Merge: %s", workspace.Name)

	var successfulMerges []string

	// Execute merge for each repository
	for _, candidate := range candidates {
		output.PrintInfo("Processing repository: %s", candidate.Repository.Name)

		if err := mergeRepository(ctx, candidate); err != nil {
			output.PrintError("Failed to merge repository %s: %v", candidate.Repository.Name, err)

			// Rollback successful merges
			if len(successfulMerges) > 0 {
				output.PrintWarning("Rolling back successful merges due to failure...")
				rollbackMerges(ctx, workspace, successfulMerges)
			}

			return errors.Wrapf(err, "merge failed for repository %s", candidate.Repository.Name)
		}

		successfulMerges = append(successfulMerges, candidate.Repository.Name)
		output.PrintSuccess("✓ Successfully merged %s", candidate.Repository.Name)
	}

	output.PrintSuccess("All repositories merged successfully!")

	// Delete workspace if requested
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

func mergeRepository(ctx context.Context, candidate MergeCandidate) error {
	repoPath := candidate.WorktreePath

	log.Debug().
		Str("repository", candidate.Repository.Name).
		Str("repoPath", repoPath).
		Str("currentBranch", candidate.CurrentBranch).
		Str("baseBranch", candidate.BaseBranch).
		Msg("Starting repository merge")

	// Step 1: Fetch latest changes
	output.PrintInfo("  Fetching latest changes...")
	if err := executeGitCommand(ctx, repoPath, "git", "fetch", "origin"); err != nil {
		return errors.Wrap(err, "failed to fetch latest changes")
	}

	// Step 2: Switch to base branch
	output.PrintInfo("  Switching to base branch: %s", candidate.BaseBranch)
	if err := executeGitCommand(ctx, repoPath, "git", "checkout", candidate.BaseBranch); err != nil {
		return errors.Wrapf(err, "failed to switch to base branch %s", candidate.BaseBranch)
	}

	// Step 3: Pull latest base branch changes
	output.PrintInfo("  Pulling latest base branch changes...")
	if err := executeGitCommand(ctx, repoPath, "git", "pull", "origin", candidate.BaseBranch); err != nil {
		return errors.Wrapf(err, "failed to pull latest changes for %s", candidate.BaseBranch)
	}

	// Step 4: Merge workspace branch
	output.PrintInfo("  Merging %s into %s...", candidate.CurrentBranch, candidate.BaseBranch)
	if err := executeGitCommand(ctx, repoPath, "git", "merge", candidate.CurrentBranch); err != nil {
		// Check if this is a merge conflict
		if isGitMergeConflict(err) {
			return errors.Errorf("merge conflict detected in %s. Please resolve conflicts manually and retry", candidate.Repository.Name)
		}
		return errors.Wrapf(err, "failed to merge %s into %s", candidate.CurrentBranch, candidate.BaseBranch)
	}

	// Step 5: Push merged changes
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

func rollbackMerges(ctx context.Context, workspace *wsm.Workspace, successfulMerges []string) {
	output.PrintWarning("🔄 Rolling back %d successful merges...", len(successfulMerges))

	for _, repoName := range successfulMerges {
		repoPath := filepath.Join(workspace.Path, repoName)

		output.PrintInfo("  Rolling back %s...", repoName)

		// Reset base branch to origin state
		if err := executeGitCommand(ctx, repoPath, "git", "checkout", workspace.BaseBranch); err != nil {
			output.PrintWarning("    Failed to checkout %s: %v", workspace.BaseBranch, err)
			continue
		}

		if err := executeGitCommand(ctx, repoPath, "git", "reset", "--hard", "origin/"+workspace.BaseBranch); err != nil {
			output.PrintWarning("    Failed to reset %s: %v", workspace.BaseBranch, err)
			continue
		}

		// Switch back to workspace branch
		if err := executeGitCommand(ctx, repoPath, "git", "checkout", workspace.Branch); err != nil {
			output.PrintWarning("    Failed to checkout %s: %v", workspace.Branch, err)
		}

		output.PrintInfo("    ✓ Rolled back %s", repoName)
	}

	output.PrintInfo("🔄 Rollback completed")
}
