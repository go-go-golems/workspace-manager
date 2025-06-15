package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// NewRemoveCommand creates the remove command
func NewRemoveCommand() *cobra.Command {
	var force bool
	var removeFiles bool

	cmd := &cobra.Command{
		Use:   "remove <workspace-name> <repo-name>",
		Short: "Remove a repository from an existing workspace",
		Long: `Remove a repository from an existing workspace and clean up its worktree.

This command:
- Loads the specified workspace configuration
- Removes the specified repository's worktree using git worktree remove
- Updates the workspace configuration to exclude the repository
- Updates go.work file if the workspace has Go repositories
- Optionally removes the repository directory from the workspace

Examples:
  # Remove a repository from a workspace
  workspace-manager remove my-feature my-old-repo

  # Force remove a repository (removes worktree even with uncommitted changes)
  workspace-manager remove my-feature my-old-repo --force

  # Remove repository and its directory from workspace
  workspace-manager remove my-feature my-old-repo --remove-files`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := args[0]
			repoName := args[1]

			wm, err := NewWorkspaceManager()
			if err != nil {
				return errors.Wrap(err, "failed to create workspace manager")
			}

			return wm.RemoveRepositoryFromWorkspace(cmd.Context(), workspaceName, repoName, force, removeFiles)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force remove worktree even with uncommitted changes")
	cmd.Flags().BoolVar(&removeFiles, "remove-files", false, "Remove the repository directory from workspace")

	return cmd
}

// RemoveRepositoryFromWorkspace removes a repository from an existing workspace
func (wm *WorkspaceManager) RemoveRepositoryFromWorkspace(ctx context.Context, workspaceName, repoName string, force, removeFiles bool) error {
	log.Info().
		Str("workspace", workspaceName).
		Str("repo", repoName).
		Bool("force", force).
		Bool("removeFiles", removeFiles).
		Msg("Removing repository from workspace")

	// Load existing workspace
	workspace, err := wm.LoadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	// Find the repository in the workspace
	var repoIndex = -1
	var targetRepo Repository
	for i, repo := range workspace.Repositories {
		if repo.Name == repoName {
			repoIndex = i
			targetRepo = repo
			break
		}
	}

	if repoIndex == -1 {
		return errors.Errorf("repository '%s' not found in workspace '%s'", repoName, workspaceName)
	}

	fmt.Printf("Removing repository '%s' from workspace '%s'\n", repoName, workspaceName)
	fmt.Printf("Repository path: %s\n", targetRepo.Path)
	fmt.Printf("Workspace path: %s\n", workspace.Path)

	// Remove the worktree
	worktreePath := filepath.Join(workspace.Path, repoName)
	if err := wm.removeWorktreeForRepo(ctx, targetRepo, worktreePath, force); err != nil {
		return errors.Wrapf(err, "failed to remove worktree for repository '%s'", repoName)
	}

	// Remove repository directory if requested
	if removeFiles {
		if _, err := os.Stat(worktreePath); err == nil {
			fmt.Printf("Removing repository directory: %s\n", worktreePath)
			if err := os.RemoveAll(worktreePath); err != nil {
				return errors.Wrapf(err, "failed to remove repository directory: %s", worktreePath)
			}
			fmt.Printf("✓ Successfully removed repository directory\n")
		}
	}

	// Remove repository from workspace configuration
	workspace.Repositories = append(workspace.Repositories[:repoIndex], workspace.Repositories[repoIndex+1:]...)

	// Update go.work file if this is a Go workspace
	if workspace.GoWorkspace {
		if err := wm.createGoWorkspace(workspace); err != nil {
			log.Warn().Err(err).Msg("Failed to update go.work file, but continuing")
			fmt.Printf("⚠️  Warning: Failed to update go.work file: %v\n", err)
		}
	}

	// Save updated workspace configuration
	if err := wm.saveWorkspace(workspace); err != nil {
		return errors.Wrap(err, "failed to save updated workspace configuration")
	}

	fmt.Printf("✓ Successfully removed repository '%s' from workspace '%s'\n", repoName, workspaceName)
	return nil
}

// removeWorktreeForRepo removes a worktree for a specific repository
func (wm *WorkspaceManager) removeWorktreeForRepo(ctx context.Context, repo Repository, worktreePath string, force bool) error {
	log.Info().
		Str("repo", repo.Name).
		Str("worktree", worktreePath).
		Bool("force", force).
		Msg("Removing worktree for repository")

	fmt.Printf("\n--- Removing worktree for %s ---\n", repo.Name)
	fmt.Printf("Worktree path: %s\n", worktreePath)

	// Check if worktree path exists
	if stat, err := os.Stat(worktreePath); os.IsNotExist(err) {
		fmt.Printf("⚠️  Worktree directory does not exist, skipping worktree removal\n")
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "error checking worktree path: %s", worktreePath)
	} else {
		fmt.Printf("✓ Worktree directory exists (type: %s)\n", map[bool]string{true: "directory", false: "file"}[stat.IsDir()])
	}

	// First, list current worktrees for debugging
	fmt.Printf("\nCurrent worktrees for %s:\n", repo.Name)
	listCmd := exec.CommandContext(ctx, "git", "worktree", "list")
	listCmd.Dir = repo.Path
	if output, err := listCmd.CombinedOutput(); err != nil {
		fmt.Printf("⚠️  Failed to list worktrees: %v\n", err)
	} else {
		fmt.Printf("%s", string(output))
	}

	// Remove worktree using git command
	var cmd *exec.Cmd
	var cmdStr string
	if force {
		cmd = exec.CommandContext(ctx, "git", "worktree", "remove", "--force", worktreePath)
		cmdStr = fmt.Sprintf("git worktree remove --force %s", worktreePath)
	} else {
		cmd = exec.CommandContext(ctx, "git", "worktree", "remove", worktreePath)
		cmdStr = fmt.Sprintf("git worktree remove %s", worktreePath)
	}
	cmd.Dir = repo.Path

	log.Info().
		Str("repo", repo.Name).
		Str("repoPath", repo.Path).
		Str("worktreePath", worktreePath).
		Str("command", cmdStr).
		Msg("Executing git worktree remove command")

	fmt.Printf("Executing: %s (in %s)\n", cmdStr, repo.Path)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().
			Err(err).
			Str("output", string(output)).
			Str("repo", repo.Name).
			Str("repoPath", repo.Path).
			Str("worktree", worktreePath).
			Str("command", cmdStr).
			Msg("Failed to remove worktree with git command")

		fmt.Printf("❌ Command failed: %s\n", cmdStr)
		fmt.Printf("   Error: %v\n", err)
		fmt.Printf("   Output: %s\n", string(output))

		return errors.Wrapf(err, "failed to remove worktree: %s", string(output))
	}

	log.Info().
		Str("output", string(output)).
		Str("repo", repo.Name).
		Str("command", cmdStr).
		Msg("Successfully removed worktree")

	fmt.Printf("✓ Successfully executed: %s\n", cmdStr)
	if len(output) > 0 {
		fmt.Printf("  Output: %s\n", string(output))
	}

	// Verify worktree was removed
	fmt.Printf("\nVerification: Remaining worktrees for %s:\n", repo.Name)
	listCmd = exec.CommandContext(ctx, "git", "worktree", "list")
	listCmd.Dir = repo.Path
	if output, err := listCmd.CombinedOutput(); err != nil {
		fmt.Printf("⚠️  Failed to list worktrees: %v\n", err)
	} else {
		fmt.Printf("%s", string(output))
	}

	return nil
}
