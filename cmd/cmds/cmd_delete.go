package cmds

import (
	"context"
	"fmt"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/charmbracelet/huh"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
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

	carapace.Gen(cmd).PositionalCompletion(WorkspaceNameCompletion())

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
	output.PrintHeader("Current workspace status")
	checker := wsm.NewStatusChecker()
	status, err := checker.GetWorkspaceStatus(ctx, workspace)
	if err == nil {
		if err := printStatusDetailed(status, false); err != nil {
			output.PrintError("Error showing status: %v", err)
		}
	} else {
		output.PrintError("Error getting status: %v", err)
	}
	fmt.Printf("\n")

	// Show what will be deleted
	if outputFormat == "json" {
		return wsm.PrintJSON(workspace)
	}

	output.PrintHeader("Workspace: %s", workspace.Name)
	fmt.Printf("  Path: %s\n", workspace.Path)
	fmt.Printf("  Repositories: %d\n", len(workspace.Repositories))

	output.PrintWarning("This will:")
	if forceWorktrees {
		fmt.Printf("  1. Remove git worktrees (git worktree remove --force)\n")
	} else {
		fmt.Printf("  1. Remove git worktrees (git worktree remove)\n")
		output.PrintWarning("     Will fail if there are uncommitted changes")
	}

	if removeFiles {
		output.PrintError("  2. DELETE the workspace directory and ALL its contents!")
		fmt.Printf("     📁 This includes: go.work, AGENT.md, and all repository worktrees\n")
	} else {
		fmt.Printf("  2. Remove workspace configuration\n")
		fmt.Printf("  3. Clean up workspace-specific files (go.work, AGENT.md)\n")
		fmt.Printf("  4. Repository worktrees will remain at: %s\n", workspace.Path)
	}

	// Confirm deletion unless forced
	if !force {
		var confirmed bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Are you sure you want to delete workspace '%s'?", workspaceName)).
					Description("This action cannot be undone.").
					Value(&confirmed),
			),
		)

		err := form.Run()
		if err != nil {
			// Check if user cancelled/aborted the form
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "user aborted") ||
				strings.Contains(errMsg, "cancelled") ||
				strings.Contains(errMsg, "aborted") ||
				strings.Contains(errMsg, "interrupt") {
				output.PrintInfo("Operation cancelled.")
				return nil
			}
			return errors.Wrap(err, "confirmation failed")
		}

		if !confirmed {
			output.PrintInfo("Operation cancelled.")
			return nil
		}
	}

	// Perform deletion
	if err := manager.DeleteWorkspace(ctx, workspaceName, removeFiles, forceWorktrees); err != nil {
		return errors.Wrap(err, "failed to delete workspace")
	}

	if removeFiles {
		output.PrintSuccess("Workspace '%s' and all files deleted successfully", workspaceName)
	} else {
		output.PrintSuccess("Workspace configuration '%s' deleted successfully", workspaceName)
		output.PrintInfo("Files remain at: %s", workspace.Path)
	}

	return nil
}
