package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"os"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize workspace repositories",
		Long: `Synchronize all repositories in the workspace with their remotes.
Supports pulling latest changes and pushing local commits.`,
	}

	cmd.AddCommand(
		NewSyncPullCommand(),
		NewSyncPushCommand(),
		NewSyncAllCommand(),
	)

	return cmd
}

func NewSyncAllCommand() *cobra.Command {
	var (
		pull   bool
		push   bool
		rebase bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Sync all repositories (pull and push)",
		Long:  "Synchronize all repositories by pulling latest changes and pushing local commits.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSyncAll(cmd.Context(), pull, push, rebase, dryRun)
		},
	}

	cmd.Flags().BoolVar(&pull, "pull", true, "Pull latest changes")
	cmd.Flags().BoolVar(&push, "push", true, "Push local commits")
	cmd.Flags().BoolVar(&rebase, "rebase", false, "Use rebase when pulling")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done")

	return cmd
}

func NewSyncPullCommand() *cobra.Command {
	var (
		rebase bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull latest changes from all repositories",
		Long:  "Pull latest changes from remote repositories in the workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSyncPull(cmd.Context(), rebase, dryRun)
		},
	}

	cmd.Flags().BoolVar(&rebase, "rebase", false, "Use rebase instead of merge")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done")

	return cmd
}

func NewSyncPushCommand() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push local commits from all repositories",
		Long:  "Push local commits to remote repositories in the workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSyncPush(cmd.Context(), dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done")

	return cmd
}

func runSyncAll(ctx context.Context, pull, push, rebase, dryRun bool) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	syncOps := wsm.NewSyncOperations(workspace)
	options := &wsm.SyncOptions{
		Pull:   pull,
		Push:   push,
		Rebase: rebase,
		DryRun: dryRun,
	}

	output.PrintHeader("Synchronizing workspace: %s", workspace.Name)
	if dryRun {
		output.PrintInfo("Dry run mode - no changes will be made")
	}

	results, err := syncOps.SyncWorkspace(ctx, options)
	if err != nil {
		return errors.Wrap(err, "sync failed")
	}

	return printSyncResults(results, dryRun)
}

func runSyncPull(ctx context.Context, rebase, dryRun bool) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	syncOps := wsm.NewSyncOperations(workspace)
	options := &wsm.SyncOptions{
		Pull:   true,
		Push:   false,
		Rebase: rebase,
		DryRun: dryRun,
	}

	output.PrintHeader("Pulling changes for workspace: %s", workspace.Name)
	if dryRun {
		output.PrintInfo("Dry run mode - no changes will be made")
	}

	results, err := syncOps.SyncWorkspace(ctx, options)
	if err != nil {
		return errors.Wrap(err, "pull failed")
	}

	return printSyncResults(results, dryRun)
}

func runSyncPush(ctx context.Context, dryRun bool) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	syncOps := wsm.NewSyncOperations(workspace)
	options := &wsm.SyncOptions{
		Pull:   false,
		Push:   true,
		Rebase: false,
		DryRun: dryRun,
	}

	output.PrintHeader("📤 Pushing changes for workspace: %s", workspace.Name)
	if dryRun {
		output.PrintInfo("Dry run mode - no changes will be made")
	}

	results, err := syncOps.SyncWorkspace(ctx, options)
	if err != nil {
		return errors.Wrap(err, "push failed")
	}

	return printSyncResults(results, dryRun)
}

func printSyncResults(results []wsm.SyncResult, dryRun bool) error {
	if len(results) == 0 {
		output.PrintInfo("No repositories to sync.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() {
		if err := w.Flush(); err != nil {
			output.LogWarn(
				fmt.Sprintf("Failed to flush table writer: %v", err),
				"Failed to flush table writer",
				"error", err,
			)
		}
	}()

	fmt.Fprintln(w, "\nREPOSITORY\tSTATUS\tPULL\tPUSH\tBEFORE\tAFTER\tERROR")
	fmt.Fprintln(w, "----------\t------\t----\t----\t------\t-----\t-----")

	successCount := 0
	conflictCount := 0

	for _, result := range results {
		status := "✅"
		if !result.Success {
			status = "❌"
		} else {
			successCount++
		}

		if result.Conflicts {
			status = "⚠️"
			conflictCount++
		}

		pullStatus := "-"
		if result.Pulled {
			pullStatus = "✅"
		}

		pushStatus := "-"
		if result.Pushed {
			pushStatus = "✅"
		}

		before := fmt.Sprintf("↑%d ↓%d", result.AheadBefore, result.BehindBefore)
		after := fmt.Sprintf("↑%d ↓%d", result.AheadAfter, result.BehindAfter)

		errorMsg := result.Error
		if len(errorMsg) > 30 {
			errorMsg = errorMsg[:27] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			result.Repository,
			status,
			pullStatus,
			pushStatus,
			before,
			after,
			errorMsg,
		)
	}

	fmt.Fprintln(w)

	// Summary
	output.PrintSuccess("Summary: %d/%d repositories synced successfully", successCount, len(results))
	if conflictCount > 0 {
		output.PrintWarning("⚠️  %d repositories have conflicts", conflictCount)
		output.PrintInfo("Resolve conflicts manually and run sync again.")
	}

	return nil
}
