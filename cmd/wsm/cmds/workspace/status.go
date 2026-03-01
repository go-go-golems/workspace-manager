package workspace

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	wsmcmdcommon "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/common"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StatusCommand shows workspace status.
type StatusCommand struct {
	*cmds.CommandDescription
}

// StatusSettings stores parsed status command settings.
type StatusSettings struct {
	WorkspaceNameArg string `glazed:"workspace-name"`
	WorkspaceName    string `glazed:"workspace"`
	Short            bool   `glazed:"short"`
	Untracked        bool   `glazed:"untracked"`
	Jobs             int    `glazed:"jobs"`
	Fetch            bool   `glazed:"fetch"`
}

var _ cmds.BareCommand = &StatusCommand{}
var _ cmds.GlazeCommand = &StatusCommand{}

type statusExecutionResult struct {
	Status    *wsm.WorkspaceStatus
	Short     bool
	Untracked bool
}

func NewStatusCommand() (*StatusCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"status",
		cmds.WithShort("Show workspace status"),
		cmds.WithLong(`Show the git status of all repositories in a workspace.
If no workspace name is provided, attempts to detect the current workspace.`),
		cmds.WithFlags(
			fields.New(
				"workspace-name",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Workspace name (positional)"),
			),
			fields.New(
				"workspace",
				fields.TypeString,
				fields.WithHelp("Workspace name"),
			),
			fields.New(
				"short",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Show short status format"),
			),
			fields.New(
				"untracked",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Include untracked files"),
			),
			fields.New(
				"jobs",
				fields.TypeInteger,
				fields.WithDefault(1),
				fields.WithHelp("Maximum concurrent repositories to process"),
			),
			fields.New(
				"fetch",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Fetch origin before computing status"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &StatusCommand{CommandDescription: desc}, nil
}

func (c *StatusCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	if result.Short {
		if err := printStatusShort(result.Status, result.Untracked); err != nil {
			return err
		}
	} else {
		if err := printStatusDetailed(result.Status, result.Untracked); err != nil {
			return err
		}
	}

	return nil
}

func (c *StatusCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	for _, row := range statusToRows(result.Status, result.Untracked) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add status row")
		}
	}

	return nil
}

func (c *StatusCommand) execute(ctx context.Context, vals *values.Values) (*statusExecutionResult, error) {
	settings_ := &StatusSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode status settings")
	}

	workspaceName := settings_.WorkspaceName
	if settings_.WorkspaceNameArg != "" {
		workspaceName = settings_.WorkspaceNameArg
	}

	workflow := workflows.NewStatusWorkflow()
	status, err := workflow.GetStatus(ctx, workflows.StatusRequest{
		WorkspaceName: workspaceName,
		Jobs:          settings_.Jobs,
		Fetch:         settings_.Fetch,
	})
	if err != nil {
		return nil, err
	}

	return &statusExecutionResult{
		Status:    status,
		Short:     settings_.Short,
		Untracked: settings_.Untracked,
	}, nil
}

func statusToRows(status *wsm.WorkspaceStatus, includeUntracked bool) []types.Row {
	rows := make([]types.Row, 0, len(status.Repositories))
	for _, repoStatus := range status.Repositories {
		untrackedFiles := []string{}
		if includeUntracked {
			untrackedFiles = repoStatus.UntrackedFiles
		}

		rows = append(rows, types.NewRow(
			types.MRP("workspace", status.Workspace.Name),
			types.MRP("workspace_path", status.Workspace.Path),
			types.MRP("overall", status.Overall),
			types.MRP("repository", repoStatus.Repository.Name),
			types.MRP("repository_path", repoStatus.Repository.Path),
			types.MRP("current_branch", repoStatus.CurrentBranch),
			types.MRP("has_changes", repoStatus.HasChanges),
			types.MRP("has_conflicts", repoStatus.HasConflicts),
			types.MRP("ahead", repoStatus.Ahead),
			types.MRP("behind", repoStatus.Behind),
			types.MRP("is_merged", repoStatus.IsMerged),
			types.MRP("needs_rebase", repoStatus.NeedsRebase),
			types.MRP("staged_count", len(repoStatus.StagedFiles)),
			types.MRP("modified_count", len(repoStatus.ModifiedFiles)),
			types.MRP("untracked_count", len(repoStatus.UntrackedFiles)),
			types.MRP("staged_files", repoStatus.StagedFiles),
			types.MRP("modified_files", repoStatus.ModifiedFiles),
			types.MRP("untracked_files", untrackedFiles),
		))
	}
	return rows
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

func NewStatusCobraCommand() (*cobra.Command, error) {
	command, err := NewStatusCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build status command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
