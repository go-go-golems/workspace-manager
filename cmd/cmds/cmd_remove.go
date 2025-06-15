package cmds

import (
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
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

			wm, err := wsm.NewWorkspaceManager()
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
