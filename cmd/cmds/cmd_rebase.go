package cmds

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewRebaseCommand creates the rebase command.
func NewRebaseCommand() *cobra.Command {
	var (
		targetBranch string
		repository   string
		dryRun       bool
		interactive  bool
		jobs         int
		manual       bool
	)

	cmd := &cobra.Command{
		Use:   "rebase [repository]",
		Short: "Rebase workspace repositories",
		Long: `Rebase workspace repositories against a target branch.

By default, rebases all repositories in the workspace against the 'main' branch.
You can specify a specific repository to rebase or change the target branch.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				repository = args[0]
			}
			return runRebase(cmd.Context(), repository, targetBranch, interactive, dryRun, jobs, manual)
		},
	}

	cmd.Flags().StringVar(&targetBranch, "target", "main", "Target branch to rebase onto")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without actually rebasing")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive rebase")
	cmd.Flags().IntVar(&jobs, "jobs", 1, "Maximum concurrent repositories to process")
	cmd.Flags().BoolVar(&manual, "manual", false, "Manual mode: show suggested commands, do not execute rebase")

	cmd.AddCommand(NewRebaseStatusCommand())
	cmd.AddCommand(NewRebaseContinueCommand())
	cmd.AddCommand(NewRebaseAbortCommand())

	return cmd
}

func runRebase(ctx context.Context, repository, targetBranch string, interactive, dryRun bool, jobs int, manual bool) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}
	workflow := workflows.NewRebaseWorkflow(workspace)

	if repository != "" {
		output.PrintHeader("🔄 Rebasing repository '%s' onto '%s'", repository, targetBranch)
	} else {
		output.PrintHeader("🔄 Rebasing all repositories onto '%s'", targetBranch)
	}

	if dryRun {
		output.PrintInfo("Dry run mode - no changes will be made")
	}

	if manual {
		fmt.Println("Manual mode: use the following commands.")
		for _, command := range workflow.ManualPlan(repository, targetBranch) {
			fmt.Println(command)
		}
		return nil
	}

	results, err := workflow.Rebase(ctx, workflows.RebaseRequest{
		Repository:   repository,
		TargetBranch: targetBranch,
		Interactive:  interactive,
		DryRun:       dryRun,
		Jobs:         jobs,
	})
	if err != nil {
		return err
	}

	return printRebaseResults(results)
}

func NewRebaseStatusCommand() *cobra.Command {
	var (
		repository string
		jobs       int
	)
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show rebase status and conflicts across repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRebaseStatus(cmd.Context(), repository, jobs)
		},
	}
	cmd.Flags().StringVar(&repository, "repo", "", "Only show status for a specific repository")
	cmd.Flags().IntVar(&jobs, "jobs", 1, "Maximum concurrent repositories to process")
	return cmd
}

func runRebaseStatus(ctx context.Context, repository string, jobs int) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}
	workflow := workflows.NewRebaseWorkflow(workspace)

	rows, err := workflow.Status(ctx, repository, jobs)
	if err != nil {
		return err
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

	fmt.Fprintln(w, "REPOSITORY\tSTATE\tCONFLICTS\tERROR")
	fmt.Fprintln(w, "----------\t-----\t---------\t-----")
	for _, row := range rows {
		errStr := row.Error
		if len(errStr) > 60 {
			errStr = errStr[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", row.Repository, string(row.State), row.Conflicts, errStr)
	}
	return nil
}

func NewRebaseContinueCommand() *cobra.Command {
	var (
		repository string
		jobs       int
	)
	cmd := &cobra.Command{
		Use:   "continue",
		Short: "Continue in-progress rebases across repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRebaseAction(cmd.Context(), repository, jobs, "continue")
		},
	}
	cmd.Flags().StringVar(&repository, "repo", "", "Only continue for a specific repository")
	cmd.Flags().IntVar(&jobs, "jobs", 1, "Maximum concurrent repositories to process")
	return cmd
}

func NewRebaseAbortCommand() *cobra.Command {
	var (
		repository string
		jobs       int
	)
	cmd := &cobra.Command{
		Use:   "abort",
		Short: "Abort in-progress rebases across repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRebaseAction(cmd.Context(), repository, jobs, "abort")
		},
	}
	cmd.Flags().StringVar(&repository, "repo", "", "Only abort for a specific repository")
	cmd.Flags().IntVar(&jobs, "jobs", 1, "Maximum concurrent repositories to process")
	return cmd
}

func runRebaseAction(ctx context.Context, repository string, jobs int, mode string) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}
	workflow := workflows.NewRebaseWorkflow(workspace)

	var rows []workflows.RebaseActionRow
	switch mode {
	case "continue":
		rows, err = workflow.Continue(ctx, repository, jobs)
	case "abort":
		rows, err = workflow.Abort(ctx, repository, jobs)
	default:
		return errors.Errorf("unsupported rebase action mode: %s", mode)
	}
	if err != nil {
		return err
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

	fmt.Fprintf(w, "\nREPOSITORY\t%s\tERROR\n", strings.ToUpper(mode))
	fmt.Fprintln(w, "----------\t-----\t-----")
	for _, row := range rows {
		status := "✅"
		if !row.Success {
			status = "❌"
		}
		errStr := row.Error
		if len(errStr) > 60 {
			errStr = errStr[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", row.Repository, status, errStr)
	}
	return nil
}

func printRebaseResults(results []workflows.RebaseResult) error {
	if len(results) == 0 {
		output.PrintInfo("No repositories to rebase.")
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

	fmt.Fprintln(w, "\nREPOSITORY\tSTATUS\tTARGET\tCOMMITS BEFORE\tCOMMITS AFTER\tERROR")
	fmt.Fprintln(w, "----------\t------\t------\t--------------\t-------------\t-----")

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

		commitsBefore := "-"
		if result.CommitsBefore > 0 {
			commitsBefore = fmt.Sprintf("%d", result.CommitsBefore)
		}
		commitsAfter := "-"
		if result.CommitsAfter > 0 {
			commitsAfter = fmt.Sprintf("%d", result.CommitsAfter)
		}

		errorMsg := result.Error
		if len(errorMsg) > 30 {
			errorMsg = errorMsg[:27] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			result.Repository,
			status,
			result.TargetBranch,
			commitsBefore,
			commitsAfter,
			errorMsg,
		)
	}

	fmt.Fprintln(w)
	output.PrintSuccess("Summary: %d/%d repositories rebased successfully", successCount, len(results))
	if conflictCount > 0 {
		output.PrintWarning("%d repositories have conflicts", conflictCount)
		output.PrintInfo("Resolve conflicts manually with:")
		fmt.Println("  - Fix conflicts in the affected files")
		fmt.Println("  - git add <resolved-files>")
		fmt.Println("  - git rebase --continue")
		fmt.Println("  Or abort the rebase with: git rebase --abort")
	}

	return nil
}
