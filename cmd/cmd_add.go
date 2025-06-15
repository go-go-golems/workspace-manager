package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// NewAddCommand creates the add command
func NewAddCommand() *cobra.Command {
	var branchName string
	var forceOverwrite bool

	cmd := &cobra.Command{
		Use:   "add <workspace-name> <repo-name>",
		Short: "Add a repository to an existing workspace",
		Long: `Add a repository to an existing workspace and create the necessary branch.

This command:
- Loads the specified workspace configuration
- Finds the specified repository in the registry
- Creates a worktree for the repository using the workspace's branch
- Updates the workspace configuration to include the new repository
- Creates or updates go.work file if the workspace has Go repositories

Examples:
  # Add a repository to an existing workspace
  workspace-manager add my-feature my-new-repo

  # Add a repository with a different branch name
  workspace-manager add my-feature my-new-repo --branch feature/different-branch

  # Force overwrite if the branch already exists
  workspace-manager add my-feature my-new-repo --force`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := args[0]
			repoName := args[1]

			wm, err := NewWorkspaceManager()
			if err != nil {
				return errors.Wrap(err, "failed to create workspace manager")
			}

			return wm.AddRepositoryToWorkspace(cmd.Context(), workspaceName, repoName, branchName, forceOverwrite)
		},
	}

	cmd.Flags().StringVarP(&branchName, "branch", "b", "", "Branch name to use (defaults to workspace's branch)")
	cmd.Flags().BoolVarP(&forceOverwrite, "force", "f", false, "Force overwrite if branch already exists")

	return cmd
}

// AddRepositoryToWorkspace adds a repository to an existing workspace
func (wm *WorkspaceManager) AddRepositoryToWorkspace(ctx context.Context, workspaceName, repoName, branchName string, forceOverwrite bool) error {
	log.Info().
		Str("workspace", workspaceName).
		Str("repo", repoName).
		Str("branch", branchName).
		Bool("force", forceOverwrite).
		Msg("Adding repository to workspace")

	// Load existing workspace
	workspace, err := wm.LoadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	// Check if repository is already in workspace
	for _, repo := range workspace.Repositories {
		if repo.Name == repoName {
			return errors.Errorf("repository '%s' is already in workspace '%s'", repoName, workspaceName)
		}
	}

	// Find the repository in the registry
	repos, err := wm.findRepositories([]string{repoName})
	if err != nil {
		return errors.Wrapf(err, "failed to find repository '%s'", repoName)
	}

	if len(repos) == 0 {
		return errors.Errorf("repository '%s' not found in registry", repoName)
	}

	repo := repos[0]

	// Use the workspace's branch if no specific branch provided
	targetBranch := branchName
	if targetBranch == "" {
		targetBranch = workspace.Branch
	}

	// Create a temporary workspace with the new repository for worktree creation
	tempWorkspace := *workspace
	tempWorkspace.Branch = targetBranch
	tempWorkspace.Repositories = []Repository{repo}

	fmt.Printf("Adding repository '%s' to workspace '%s'\n", repoName, workspaceName)
	fmt.Printf("Target branch: %s\n", targetBranch)
	fmt.Printf("Workspace path: %s\n", workspace.Path)

	// Create worktree for the new repository
	if err := wm.createWorktreeForAdd(ctx, workspace, repo, targetBranch, forceOverwrite); err != nil {
		return errors.Wrapf(err, "failed to create worktree for repository '%s'", repoName)
	}

	// Add repository to workspace configuration
	workspace.Repositories = append(workspace.Repositories, repo)

	// Update go.work file if this is a Go workspace and the new repo has go.mod
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

	fmt.Printf("✓ Successfully added repository '%s' to workspace '%s'\n", repoName, workspaceName)
	return nil
}

// createWorktreeForAdd creates a worktree for adding a repository to an existing workspace
func (wm *WorkspaceManager) createWorktreeForAdd(ctx context.Context, workspace *Workspace, repo Repository, branch string, forceOverwrite bool) error {
	targetPath := filepath.Join(workspace.Path, repo.Name)

	log.Info().
		Str("repo", repo.Name).
		Str("branch", branch).
		Str("target", targetPath).
		Bool("force", forceOverwrite).
		Msg("Creating worktree for add operation")

	// Check if target path already exists
	if _, err := os.Stat(targetPath); err == nil {
		return errors.Errorf("target path '%s' already exists", targetPath)
	}

	if branch == "" {
		// No specific branch, create worktree from current branch
		return wm.executeWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", targetPath)
	}

	// Check if branch exists locally
	branchExists, err := wm.checkBranchExists(ctx, repo.Path, branch)
	if err != nil {
		return errors.Wrapf(err, "failed to check if branch %s exists", branch)
	}

	// Check if branch exists remotely
	remoteBranchExists, err := wm.checkRemoteBranchExists(ctx, repo.Path, branch)
	if err != nil {
		log.Warn().Err(err).Str("branch", branch).Msg("Could not check remote branch existence")
	}

	fmt.Printf("\nBranch status for %s:\n", repo.Name)
	fmt.Printf("  Local branch '%s' exists: %v\n", branch, branchExists)
	fmt.Printf("  Remote branch 'origin/%s' exists: %v\n", branch, remoteBranchExists)

	if branchExists {
		if forceOverwrite {
			fmt.Printf("Force overwriting branch '%s'...\n", branch)
			if remoteBranchExists {
				return wm.executeWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-B", branch, targetPath, "origin/"+branch)
			} else {
				return wm.executeWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-B", branch, targetPath)
			}
		} else {
			// Branch exists locally - ask user what to do unless force is specified
			fmt.Printf("\n⚠️  Branch '%s' already exists in repository '%s'\n", branch, repo.Name)
			fmt.Printf("What would you like to do?\n")
			fmt.Printf("  [o] Overwrite the existing branch (git worktree add -B)\n")
			fmt.Printf("  [u] Use the existing branch as-is (git worktree add)\n")
			fmt.Printf("  [c] Cancel operation\n")
			fmt.Printf("Choice [o/u/c]: ")

			var choice string
			fmt.Scanln(&choice)

			switch strings.ToLower(choice) {
			case "o", "overwrite":
				fmt.Printf("Overwriting branch '%s'...\n", branch)
				if remoteBranchExists {
					return wm.executeWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-B", branch, targetPath, "origin/"+branch)
				} else {
					return wm.executeWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-B", branch, targetPath)
				}
			case "u", "use":
				fmt.Printf("Using existing branch '%s'...\n", branch)
				return wm.executeWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", targetPath, branch)
			case "c", "cancel":
				return errors.New("operation cancelled by user")
			default:
				return errors.New("invalid choice, operation cancelled")
			}
		}
	} else {
		// Branch doesn't exist locally
		if remoteBranchExists {
			fmt.Printf("Creating worktree from remote branch origin/%s...\n", branch)
			return wm.executeWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-b", branch, targetPath, "origin/"+branch)
		} else {
			fmt.Printf("Creating new branch '%s' and worktree...\n", branch)
			return wm.executeWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-b", branch, targetPath)
		}
	}
}
