package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewDeleteCommand creates the delete command
func NewDeleteCommand() *cobra.Command {
	var (
		force          bool
		forceWorktrees bool
		removeFiles    bool
		outputFormat   string
	)

	cmd := &cobra.Command{
		Use:   "delete <workspace-name>",
		Short: "Delete a workspace",
		Long: `Delete a workspace and optionally remove its files.

This command removes the workspace configuration and optionally deletes
the workspace directory and all its contents. Use with caution.

Examples:
  # Delete workspace configuration only
  workspace-manager delete my-workspace

  # Delete workspace and all files
  workspace-manager delete my-workspace --remove-files

  # Force delete without confirmation
  workspace-manager delete my-workspace --force --remove-files

  # Force worktree removal even with uncommitted changes
  workspace-manager delete my-workspace --force-worktrees --remove-files`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), args[0], force, forceWorktrees, removeFiles, outputFormat)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force delete without confirmation")
	cmd.Flags().BoolVar(&forceWorktrees, "force-worktrees", false, "Force worktree removal even with uncommitted changes")
	cmd.Flags().BoolVar(&removeFiles, "remove-files", false, "Remove workspace files and directories")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")

	return cmd
}

func runDelete(ctx context.Context, workspaceName string, force bool, forceWorktrees bool, removeFiles bool, outputFormat string) error {
	manager, err := wsm.NewWorkspaceManager()
	if err != nil {
		return errors.Wrap(err, "failed to create workspace manager")
	}

	// Load workspace
	workspace, err := manager.LoadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "workspace '%s' not found", workspaceName)
	}

	// Show workspace status first
	fmt.Printf("Current workspace status:\n")
	fmt.Printf("========================\n")
	checker := wsm.NewStatusChecker()
	status, err := checker.GetWorkspaceStatus(ctx, workspace)
	if err == nil {
		if err := printStatusDetailed(status, false); err != nil {
			fmt.Printf("Error showing status: %v\n", err)
		}
	} else {
		fmt.Printf("Error getting status: %v\n", err)
	}
	fmt.Printf("\n")

	// Show what will be deleted
	if outputFormat == "json" {
		return wsm.PrintJSON(workspace)
	}

	fmt.Printf("Workspace: %s\n", workspace.Name)
	fmt.Printf("Path: %s\n", workspace.Path)
	fmt.Printf("Repositories: %d\n", len(workspace.Repositories))

	fmt.Printf("\nThis will:\n")
	if forceWorktrees {
		fmt.Printf("  1. Remove git worktrees (git worktree remove --force)\n")
	} else {
		fmt.Printf("  1. Remove git worktrees (git worktree remove)\n")
		fmt.Printf("     ⚠️  Will fail if there are uncommitted changes\n")
	}

	if removeFiles {
		fmt.Printf("  2. DELETE the workspace directory and ALL its contents!\n")
		fmt.Printf("     📁 This includes: go.work, AGENT.md, and all repository worktrees\n")
	} else {
		fmt.Printf("  2. Remove workspace configuration\n")
		fmt.Printf("  3. Clean up workspace-specific files (go.work, AGENT.md)\n")
		fmt.Printf("  4. Repository worktrees will remain at: %s\n", workspace.Path)
	}

	// Confirm deletion unless forced
	if !force {
		fmt.Printf("\nAre you sure you want to delete workspace '%s'? [y/N]: ", workspaceName)
		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// Perform deletion
	if err := manager.DeleteWorkspace(ctx, workspaceName, removeFiles, forceWorktrees); err != nil {
		return errors.Wrap(err, "failed to delete workspace")
	}

	if removeFiles {
		fmt.Printf("✓ Workspace '%s' and all files deleted successfully\n", workspaceName)
	} else {
		fmt.Printf("✓ Workspace configuration '%s' deleted successfully\n", workspaceName)
		fmt.Printf("Files remain at: %s\n", workspace.Path)
	}

	return nil
}
