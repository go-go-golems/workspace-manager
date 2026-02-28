package git

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
	"github.com/go-go-golems/glazed/pkg/types"
	wsmcmdcommon "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/common"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type RebaseCommand struct {
	*cmds.CommandDescription
}

type RebaseStatusCommand struct {
	*cmds.CommandDescription
}

type RebaseContinueCommand struct {
	*cmds.CommandDescription
}

type RebaseAbortCommand struct {
	*cmds.CommandDescription
}

type RebaseSettings struct {
	Repository  string `glazed:"repository"`
	Target      string `glazed:"target"`
	DryRun      bool   `glazed:"dry-run"`
	Interactive bool   `glazed:"interactive"`
	Jobs        int    `glazed:"jobs"`
	Manual      bool   `glazed:"manual"`
}

type RebaseStatusSettings struct {
	Repository string `glazed:"repo"`
	Jobs       int    `glazed:"jobs"`
}

type RebaseActionSettings struct {
	Repository string `glazed:"repo"`
	Jobs       int    `glazed:"jobs"`
}

var _ cmds.BareCommand = &RebaseCommand{}
var _ cmds.BareCommand = &RebaseStatusCommand{}
var _ cmds.BareCommand = &RebaseContinueCommand{}
var _ cmds.BareCommand = &RebaseAbortCommand{}

func NewRebaseCommand() (*RebaseCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"rebase",
		cmds.WithShort("Rebase workspace repositories"),
		cmds.WithLong(`Rebase workspace repositories against a target branch.
By default, rebases all repositories in the workspace against 'main'.`),
		cmds.WithFlags(
			fields.New("repository", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Specific repository to rebase")),
			fields.New("target", fields.TypeString, fields.WithDefault("main"), fields.WithHelp("Target branch to rebase onto")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Show what would be done without rebasing")),
			fields.New("interactive", fields.TypeBool, fields.WithShortFlag("i"), fields.WithDefault(false), fields.WithHelp("Interactive rebase")),
			fields.New("jobs", fields.TypeInteger, fields.WithDefault(1), fields.WithHelp("Maximum concurrent repositories to process")),
			fields.New("manual", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Manual mode: show suggested commands only")),
		),
	)
	if err != nil {
		return nil, err
	}
	return &RebaseCommand{CommandDescription: desc}, nil
}

func (c *RebaseCommand) Run(ctx context.Context, vals *values.Values) error {
	settings_ := &RebaseSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode rebase settings")
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}
	workflow := workflows.NewRebaseWorkflow(workspace)

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		if settings_.Repository != "" {
			output.PrintHeader("Rebasing repository '%s' onto '%s'", settings_.Repository, settings_.Target)
		} else {
			output.PrintHeader("Rebasing all repositories onto '%s'", settings_.Target)
		}
		if settings_.DryRun {
			output.PrintInfo("Dry run mode - no changes will be made")
		}
	}

	if settings_.Manual {
		commands := workflow.ManualPlan(settings_.Repository, settings_.Target)
		if wsmcmdcommon.ShouldOutputHuman(mode) {
			fmt.Println("Manual mode: use the following commands.")
			for _, command := range commands {
				fmt.Println(command)
			}
		}
		if wsmcmdcommon.ShouldOutputData(mode) {
			rows := make([]types.Row, 0, len(commands))
			for _, command := range commands {
				rows = append(rows, types.NewRow(
					types.MRP("workspace", workspace.Name),
					types.MRP("repository", settings_.Repository),
					types.MRP("target_branch", settings_.Target),
					types.MRP("manual", true),
					types.MRP("command", command),
				))
			}
			if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
				return errors.Wrap(err, "failed to emit rebase manual rows")
			}
		}
		return nil
	}

	results, err := workflow.Rebase(ctx, workflows.RebaseRequest{
		Repository:   settings_.Repository,
		TargetBranch: settings_.Target,
		Interactive:  settings_.Interactive,
		DryRun:       settings_.DryRun,
		Jobs:         settings_.Jobs,
	})
	if err != nil {
		return err
	}

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		if err := printRebaseResults(results); err != nil {
			return err
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		rows := make([]types.Row, 0, len(results))
		for _, result := range results {
			rows = append(rows, types.NewRow(
				types.MRP("workspace", workspace.Name),
				types.MRP("repository", result.Repository),
				types.MRP("success", result.Success),
				types.MRP("error", result.Error),
				types.MRP("rebased", result.Rebased),
				types.MRP("conflicts", result.Conflicts),
				types.MRP("commits_before", result.CommitsBefore),
				types.MRP("commits_after", result.CommitsAfter),
				types.MRP("target_branch", result.TargetBranch),
				types.MRP("dry_run", settings_.DryRun),
				types.MRP("interactive", settings_.Interactive),
				types.MRP("jobs", settings_.Jobs),
			))
		}
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit rebase rows")
		}
	}

	return nil
}

func NewRebaseStatusCommand() (*RebaseStatusCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"status",
		cmds.WithShort("Show rebase status and conflicts across repositories"),
		cmds.WithFlags(
			fields.New("repo", fields.TypeString, fields.WithHelp("Only show status for a specific repository")),
			fields.New("jobs", fields.TypeInteger, fields.WithDefault(1), fields.WithHelp("Maximum concurrent repositories to process")),
		),
	)
	if err != nil {
		return nil, err
	}
	return &RebaseStatusCommand{CommandDescription: desc}, nil
}

func (c *RebaseStatusCommand) Run(ctx context.Context, vals *values.Values) error {
	settings_ := &RebaseStatusSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode rebase status settings")
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}
	workflow := workflows.NewRebaseWorkflow(workspace)

	rows, err := workflow.Status(ctx, settings_.Repository, settings_.Jobs)
	if err != nil {
		return err
	}

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		if err := printRebaseStatusRows(rows); err != nil {
			return err
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		outRows := make([]types.Row, 0, len(rows))
		for _, row := range rows {
			outRows = append(outRows, types.NewRow(
				types.MRP("workspace", workspace.Name),
				types.MRP("repository", row.Repository),
				types.MRP("state", string(row.State)),
				types.MRP("conflicts", row.Conflicts),
				types.MRP("error", row.Error),
				types.MRP("jobs", settings_.Jobs),
			))
		}
		if err := wsmcmdcommon.EmitRows(ctx, vals, outRows); err != nil {
			return errors.Wrap(err, "failed to emit rebase status rows")
		}
	}

	return nil
}

func NewRebaseContinueCommand() (*RebaseContinueCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"continue",
		cmds.WithShort("Continue in-progress rebases across repositories"),
		cmds.WithFlags(
			fields.New("repo", fields.TypeString, fields.WithHelp("Only continue for a specific repository")),
			fields.New("jobs", fields.TypeInteger, fields.WithDefault(1), fields.WithHelp("Maximum concurrent repositories to process")),
		),
	)
	if err != nil {
		return nil, err
	}
	return &RebaseContinueCommand{CommandDescription: desc}, nil
}

func (c *RebaseContinueCommand) Run(ctx context.Context, vals *values.Values) error {
	return runRebaseAction(ctx, vals, "continue")
}

func NewRebaseAbortCommand() (*RebaseAbortCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"abort",
		cmds.WithShort("Abort in-progress rebases across repositories"),
		cmds.WithFlags(
			fields.New("repo", fields.TypeString, fields.WithHelp("Only abort for a specific repository")),
			fields.New("jobs", fields.TypeInteger, fields.WithDefault(1), fields.WithHelp("Maximum concurrent repositories to process")),
		),
	)
	if err != nil {
		return nil, err
	}
	return &RebaseAbortCommand{CommandDescription: desc}, nil
}

func (c *RebaseAbortCommand) Run(ctx context.Context, vals *values.Values) error {
	return runRebaseAction(ctx, vals, "abort")
}

func runRebaseAction(ctx context.Context, vals *values.Values, mode string) error {
	settings_ := &RebaseActionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrapf(err, "failed to decode rebase %s settings", mode)
	}

	outputMode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(outputMode) && !wsmcmdcommon.ShouldOutputData(outputMode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(outputMode)
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}
	workflow := workflows.NewRebaseWorkflow(workspace)

	var rows []workflows.RebaseActionRow
	switch mode {
	case "continue":
		rows, err = workflow.Continue(ctx, settings_.Repository, settings_.Jobs)
	case "abort":
		rows, err = workflow.Abort(ctx, settings_.Repository, settings_.Jobs)
	default:
		return errors.Errorf("unsupported rebase action mode: %s", mode)
	}
	if err != nil {
		return err
	}

	if wsmcmdcommon.ShouldOutputHuman(outputMode) {
		if err := printRebaseActionRows(mode, rows); err != nil {
			return err
		}
	}

	if wsmcmdcommon.ShouldOutputData(outputMode) {
		outRows := make([]types.Row, 0, len(rows))
		for _, row := range rows {
			outRows = append(outRows, types.NewRow(
				types.MRP("workspace", workspace.Name),
				types.MRP("mode", mode),
				types.MRP("repository", row.Repository),
				types.MRP("success", row.Success),
				types.MRP("error", row.Error),
				types.MRP("jobs", settings_.Jobs),
			))
		}
		if err := wsmcmdcommon.EmitRows(ctx, vals, outRows); err != nil {
			return errors.Wrapf(err, "failed to emit rebase %s rows", mode)
		}
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
			result.Repository, status, result.TargetBranch, commitsBefore, commitsAfter, errorMsg)
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

func printRebaseStatusRows(rows []workflows.RebaseStatusRow) error {
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

func printRebaseActionRows(mode string, rows []workflows.RebaseActionRow) error {
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

func NewRebaseCobraCommand() (*cobra.Command, error) {
	rebaseCmd, err := NewRebaseCommand()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase command")
	}
	statusCmd, err := NewRebaseStatusCommand()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase status command")
	}
	continueCmd, err := NewRebaseContinueCommand()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase continue command")
	}
	abortCmd, err := NewRebaseAbortCommand()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase abort command")
	}

	rebaseCobra, err := wsmcmdcommon.BuildCobraCommand(rebaseCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase cobra command")
	}
	statusCobra, err := wsmcmdcommon.BuildCobraCommand(statusCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase status cobra command")
	}
	continueCobra, err := wsmcmdcommon.BuildCobraCommand(continueCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase continue cobra command")
	}
	abortCobra, err := wsmcmdcommon.BuildCobraCommand(abortCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase abort cobra command")
	}

	rebaseCobra.AddCommand(statusCobra, continueCobra, abortCobra)
	return rebaseCobra, nil
}
