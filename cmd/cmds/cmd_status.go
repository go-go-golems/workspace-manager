package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/carapace-sh/carapace"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewStatusCommand() *cobra.Command {
	var (
		short     bool
		untracked bool
		workspace string
		jobs      int
	)

	cmd := &cobra.Command{
		Use:   "status [workspace-name]",
		Short: "Show workspace status",
		Long: `Show the git status of all repositories in a workspace.
If no workspace name is provided, attempts to detect the current workspace.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := workspace
			if len(args) > 0 {
				workspaceName = args[0]
			}
			return runStatus(cmd.Context(), workspaceName, short, untracked, jobs)
		},
	}

	cmd.Flags().BoolVar(&short, "short", false, "Show short status format")
	cmd.Flags().BoolVar(&untracked, "untracked", false, "Include untracked files")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	cmd.Flags().IntVar(&jobs, "jobs", 1, "Maximum concurrent repositories to process")

	carapace.Gen(cmd).PositionalCompletion(WorkspaceNameCompletion())

	return cmd
}

func runStatus(ctx context.Context, workspaceName string, short, untracked bool, jobs int) error {
	// If no workspace specified, try to detect current workspace
	if workspaceName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to get current directory")
		}

		detected, err := detectWorkspace(cwd)
		if err != nil {
			return errors.Wrap(err, "failed to detect workspace. Use 'workspace-manager status <workspace-name>' or specify --workspace flag")
		}
		workspaceName = detected
	}

	// Load workspace
	workspace, err := loadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	// Get status
	checker := wsm.NewStatusChecker()
	status, err := checker.GetWorkspaceStatusWithOptions(ctx, workspace, wsm.StatusOptions{MaxJobs: jobs})
	if err != nil {
		return errors.Wrap(err, "failed to get workspace status")
	}

	// Display status
	if short {
		return printStatusShort(status, untracked)
	}

	return printStatusDetailed(status, untracked)
}

func printStatusShort(status *wsm.WorkspaceStatus, includeUntracked bool) error {
	output.PrintHeader("Workspace: %s (%s)", status.Workspace.Name, status.Overall)

	for _, repoStatus := range status.Repositories {
		symbol := getRepositoryStatusSymbol(repoStatus)
		fmt.Printf("%s %s", symbol, repoStatus.Repository.Name)

		if repoStatus.CurrentBranch != "" {
			fmt.Printf(" [%s]", repoStatus.CurrentBranch)
		}

		if repoStatus.Ahead > 0 || repoStatus.Behind > 0 {
			fmt.Printf(" ↑%d ↓%d", repoStatus.Ahead, repoStatus.Behind)
		}

		changes := []string{}
		if len(repoStatus.StagedFiles) > 0 {
			changes = append(changes, fmt.Sprintf("S:%d", len(repoStatus.StagedFiles)))
		}
		if len(repoStatus.ModifiedFiles) > 0 {
			changes = append(changes, fmt.Sprintf("M:%d", len(repoStatus.ModifiedFiles)))
		}
		if includeUntracked && len(repoStatus.UntrackedFiles) > 0 {
			changes = append(changes, fmt.Sprintf("U:%d", len(repoStatus.UntrackedFiles)))
		}

		if len(changes) > 0 {
			fmt.Printf(" [%s]", strings.Join(changes, " "))
		}

		fmt.Println()
	}

	return nil
}

func printStatusDetailed(status *wsm.WorkspaceStatus, includeUntracked bool) error {
	output.PrintHeader("Workspace: %s", status.Workspace.Name)
	output.PrintInfo("Path: %s", status.Workspace.Path)
	output.PrintInfo("Overall Status: %s", status.Overall)
	fmt.Println()

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

	fmt.Fprintln(w, "REPOSITORY\tBRANCH\tSTATUS\tCHANGES\tSYNC\tMERGED\tREBASE")
	fmt.Fprintln(w, "----------\t------\t------\t-------\t----\t------\t------")

	for _, repoStatus := range status.Repositories {
		repoName := repoStatus.Repository.Name
		branch := repoStatus.CurrentBranch
		if branch == "" {
			branch = "-"
		}

		statusStr := getStatusString(repoStatus)
		changesStr := getChangesString(repoStatus, includeUntracked)
		syncStr := getSyncString(repoStatus)
		mergedStr := getMergedString(repoStatus)
		rebaseStr := getRebaseString(repoStatus)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			repoName, branch, statusStr, changesStr, syncStr, mergedStr, rebaseStr)
	}

	fmt.Fprintln(w)

	// Show detailed changes if any
	for _, repoStatus := range status.Repositories {
		if repoStatus.HasChanges || (includeUntracked && len(repoStatus.UntrackedFiles) > 0) {
			fmt.Printf("\n%s:\n", repoStatus.Repository.Name)

			if len(repoStatus.StagedFiles) > 0 {
				fmt.Printf("  Staged files:\n")
				for _, file := range repoStatus.StagedFiles {
					fmt.Printf("    + %s\n", file)
				}
			}

			if len(repoStatus.ModifiedFiles) > 0 {
				fmt.Printf("  Modified files:\n")
				for _, file := range repoStatus.ModifiedFiles {
					fmt.Printf("    M %s\n", file)
				}
			}

			if includeUntracked && len(repoStatus.UntrackedFiles) > 0 {
				fmt.Printf("  Untracked files:\n")
				for _, file := range repoStatus.UntrackedFiles {
					fmt.Printf("    ? %s\n", file)
				}
			}
		}
	}

	return nil
}

func getRepositoryStatusSymbol(status wsm.RepositoryStatus) string {
	if status.HasConflicts {
		return "⚠️ "
	}
	if status.HasChanges {
		return "🔄"
	}
	if status.Ahead > 0 || status.Behind > 0 {
		return "📤"
	}
	return "✅"
}

func getStatusString(status wsm.RepositoryStatus) string {
	if status.HasConflicts {
		return "conflict"
	}
	if status.HasChanges {
		return "modified"
	}
	return "clean"
}

func getChangesString(status wsm.RepositoryStatus, includeUntracked bool) string {
	parts := []string{}

	if len(status.StagedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("S:%d", len(status.StagedFiles)))
	}
	if len(status.ModifiedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("M:%d", len(status.ModifiedFiles)))
	}
	if includeUntracked && len(status.UntrackedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("U:%d", len(status.UntrackedFiles)))
	}

	if len(parts) == 0 {
		return "-"
	}

	return strings.Join(parts, " ")
}

func getSyncString(status wsm.RepositoryStatus) string {
	if status.Ahead == 0 && status.Behind == 0 {
		return "✓"
	}
	return fmt.Sprintf("↑%d ↓%d", status.Ahead, status.Behind)
}

func getMergedString(status wsm.RepositoryStatus) string {
	if status.IsMerged {
		return "✓"
	}
	return "-"
}

func getRebaseString(status wsm.RepositoryStatus) string {
	if status.NeedsRebase {
		return "⚠️"
	}
	return "✓"
}
