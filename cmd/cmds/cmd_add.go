package cmds

import (
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
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

			wm, err := wsm.NewWorkspaceManager()
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
