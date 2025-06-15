package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"os"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewBranchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage branches across workspace repositories",
		Long: `Create, switch, and manage branches across all repositories in the workspace.
This ensures consistent branch operations across your multi-repository development.`,
	}

	cmd.AddCommand(
		NewBranchCreateCommand(),
		NewBranchSwitchCommand(),
		NewBranchListCommand(),
	)

	return cmd
}

func NewBranchCreateCommand() *cobra.Command {
	var track bool

	cmd := &cobra.Command{
		Use:   "create [branch-name]",
		Short: "Create a branch across all repositories",
		Long:  "Create a new branch with the same name across all repositories in the workspace.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBranchCreate(cmd.Context(), args[0], track)
		},
	}

	cmd.Flags().BoolVar(&track, "track", false, "Set up tracking for the new branch")

	return cmd
}

func NewBranchSwitchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch [branch-name]",
		Short: "Switch to a branch across all repositories",
		Long:  "Switch all repositories in the workspace to the specified branch.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBranchSwitch(cmd.Context(), args[0])
		},
	}

	return cmd
}

func NewBranchListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List current branches across repositories",
		Long:  "Show the current branch for each repository in the workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBranchList(cmd.Context())
		},
	}

	return cmd
}

func runBranchCreate(ctx context.Context, branchName string, track bool) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	syncOps := wsm.NewSyncOperations(workspace)

	fmt.Printf("🌿 Creating branch '%s' across workspace: %s\n", branchName, workspace.Name)

	results, err := syncOps.CreateBranch(ctx, branchName, track)
	if err != nil {
		return errors.Wrap(err, "branch creation failed")
	}

	return printBranchResults(results, "create")
}

func runBranchSwitch(ctx context.Context, branchName string) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	syncOps := wsm.NewSyncOperations(workspace)

	fmt.Printf("🔄 Switching to branch '%s' across workspace: %s\n", branchName, workspace.Name)

	results, err := syncOps.SwitchBranch(ctx, branchName)
	if err != nil {
		return errors.Wrap(err, "branch switch failed")
	}

	return printBranchResults(results, "switch")
}

func runBranchList(ctx context.Context) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	fmt.Printf("📋 Current branches in workspace: %s\n", workspace.Name)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "\nREPOSITORY\tCURRENT BRANCH\tSTATUS")
	fmt.Fprintln(w, "----------\t--------------\t------")

	checker := wsm.NewStatusChecker()
	for _, repo := range workspace.Repositories {
		// Get current workspace status for this repo
		status, err := checker.GetWorkspaceStatus(ctx, &wsm.Workspace{
			Path:         workspace.Path,
			Repositories: []wsm.Repository{repo},
		})
		if err != nil {
			fmt.Fprintf(w, "%s\t%s\t%s\n", repo.Name, "unknown", "❌")
			continue
		}

		if len(status.Repositories) > 0 {
			repoStatus := status.Repositories[0]
			statusSymbol := "✅"
			if repoStatus.HasChanges {
				statusSymbol = "🔄"
			}
			if repoStatus.HasConflicts {
				statusSymbol = "⚠️"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\n",
				repo.Name,
				repoStatus.CurrentBranch,
				statusSymbol,
			)
		}
	}

	fmt.Fprintln(w)
	return nil
}

func printBranchResults(results []wsm.SyncResult, operation string) error {
	if len(results) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "\nREPOSITORY\tSTATUS\tERROR")
	fmt.Fprintln(w, "----------\t------\t-----")

	successCount := 0

	for _, result := range results {
		status := "✅"
		if !result.Success {
			status = "❌"
		} else {
			successCount++
		}

		errorMsg := result.Error
		if len(errorMsg) > 50 {
			errorMsg = errorMsg[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			result.Repository,
			status,
			errorMsg,
		)
	}

	fmt.Fprintln(w)

	// Summary
	fmt.Printf("Summary: %d/%d repositories %s successfully\n", successCount, len(results), operation)

	if successCount < len(results) {
		fmt.Printf("\n⚠️  Some repositories failed. Check errors above and resolve manually.\n")
	}

	return nil
}
