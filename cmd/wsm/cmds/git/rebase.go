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
	"github.com/go-go-golems/glazed/pkg/middlewares"
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

type rebaseExecutionResult struct {
	WorkspaceName string
	Repository    string
	TargetBranch  string
	DryRun        bool
	Interactive   bool
	Jobs          int
	Manual        bool
	Commands      []string
	Results       []workflows.RebaseResult
}

type rebaseStatusExecutionResult struct {
	WorkspaceName string
	Jobs          int
	Rows          []workflows.RebaseStatusRow
}

type rebaseActionExecutionResult struct {
	WorkspaceName string
	Mode          string
	Jobs          int
	Rows          []workflows.RebaseActionRow
}

var _ cmds.BareCommand = &RebaseCommand{}
var _ cmds.GlazeCommand = &RebaseCommand{}
var _ cmds.BareCommand = &RebaseStatusCommand{}
var _ cmds.GlazeCommand = &RebaseStatusCommand{}
var _ cmds.BareCommand = &RebaseContinueCommand{}
var _ cmds.GlazeCommand = &RebaseContinueCommand{}
var _ cmds.BareCommand = &RebaseAbortCommand{}
var _ cmds.GlazeCommand = &RebaseAbortCommand{}

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
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	if result.Repository != "" {
		output.PrintHeader("Rebasing repository '%s' onto '%s'", result.Repository, result.TargetBranch)
	} else {
		output.PrintHeader("Rebasing all repositories onto '%s'", result.TargetBranch)
	}
	if result.DryRun {
		output.PrintInfo("Dry run mode - no changes will be made")
	}

	if result.Manual {
		fmt.Println("Manual mode: use the following commands.")
		for _, command := range result.Commands {
			fmt.Println(command)
		}
		return nil
	}

	return printRebaseResults(result.Results)
}

func (c *RebaseCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	for _, row := range rebaseResultToRows(result) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add rebase row")
		}
	}

	return nil
}

func (c *RebaseCommand) execute(ctx context.Context, vals *values.Values) (*rebaseExecutionResult, error) {
	settings_ := &RebaseSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode rebase settings")
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect current workspace")
	}
	workflow := workflows.NewRebaseWorkflow(workspace)

	if settings_.Manual {
		commands := workflow.ManualPlan(settings_.Repository, settings_.Target)
		return &rebaseExecutionResult{
			WorkspaceName: workspace.Name,
			Repository:    settings_.Repository,
			TargetBranch:  settings_.Target,
			DryRun:        settings_.DryRun,
			Interactive:   settings_.Interactive,
			Jobs:          settings_.Jobs,
			Manual:        true,
			Commands:      commands,
		}, nil
	}

	results, err := workflow.Rebase(ctx, workflows.RebaseRequest{
		Repository:   settings_.Repository,
		TargetBranch: settings_.Target,
		Interactive:  settings_.Interactive,
		DryRun:       settings_.DryRun,
		Jobs:         settings_.Jobs,
	})
	if err != nil {
		return nil, err
	}

	return &rebaseExecutionResult{
		WorkspaceName: workspace.Name,
		Repository:    settings_.Repository,
		TargetBranch:  settings_.Target,
		DryRun:        settings_.DryRun,
		Interactive:   settings_.Interactive,
		Jobs:          settings_.Jobs,
		Manual:        false,
		Results:       results,
	}, nil
}

func rebaseResultToRows(result *rebaseExecutionResult) []types.Row {
	if result.Manual {
		rows := make([]types.Row, 0, len(result.Commands))
		for _, command := range result.Commands {
			rows = append(rows, types.NewRow(
				types.MRP("workspace", result.WorkspaceName),
				types.MRP("repository", result.Repository),
				types.MRP("target_branch", result.TargetBranch),
				types.MRP("manual", true),
				types.MRP("command", command),
			))
		}
		return rows
	}

	rows := make([]types.Row, 0, len(result.Results))
	for _, row := range result.Results {
		rows = append(rows, types.NewRow(
			types.MRP("workspace", result.WorkspaceName),
			types.MRP("repository", row.Repository),
			types.MRP("success", row.Success),
			types.MRP("error", row.Error),
			types.MRP("rebased", row.Rebased),
			types.MRP("conflicts", row.Conflicts),
			types.MRP("commits_before", row.CommitsBefore),
			types.MRP("commits_after", row.CommitsAfter),
			types.MRP("target_branch", row.TargetBranch),
			types.MRP("dry_run", result.DryRun),
			types.MRP("interactive", result.Interactive),
			types.MRP("jobs", result.Jobs),
		))
	}

	return rows
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
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	return printRebaseStatusRows(result.Rows)
}

func (c *RebaseStatusCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	for _, row := range rebaseStatusResultToRows(result) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add rebase status row")
		}
	}

	return nil
}

func (c *RebaseStatusCommand) execute(ctx context.Context, vals *values.Values) (*rebaseStatusExecutionResult, error) {
	settings_ := &RebaseStatusSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode rebase status settings")
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect current workspace")
	}
	workflow := workflows.NewRebaseWorkflow(workspace)

	rows, err := workflow.Status(ctx, settings_.Repository, settings_.Jobs)
	if err != nil {
		return nil, err
	}

	return &rebaseStatusExecutionResult{
		WorkspaceName: workspace.Name,
		Jobs:          settings_.Jobs,
		Rows:          rows,
	}, nil
}

func rebaseStatusResultToRows(result *rebaseStatusExecutionResult) []types.Row {
	rows := make([]types.Row, 0, len(result.Rows))
	for _, row := range result.Rows {
		rows = append(rows, types.NewRow(
			types.MRP("workspace", result.WorkspaceName),
			types.MRP("repository", row.Repository),
			types.MRP("state", string(row.State)),
			types.MRP("conflicts", row.Conflicts),
			types.MRP("error", row.Error),
			types.MRP("jobs", result.Jobs),
		))
	}
	return rows
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
	result, err := executeRebaseAction(ctx, vals, "continue")
	if err != nil {
		return err
	}
	return printRebaseActionRows("continue", result.Rows)
}

func (c *RebaseContinueCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := executeRebaseAction(ctx, vals, "continue")
	if err != nil {
		return err
	}
	for _, row := range rebaseActionResultToRows(result) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add rebase continue row")
		}
	}
	return nil
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
	result, err := executeRebaseAction(ctx, vals, "abort")
	if err != nil {
		return err
	}
	return printRebaseActionRows("abort", result.Rows)
}

func (c *RebaseAbortCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := executeRebaseAction(ctx, vals, "abort")
	if err != nil {
		return err
	}
	for _, row := range rebaseActionResultToRows(result) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add rebase abort row")
		}
	}
	return nil
}

func executeRebaseAction(ctx context.Context, vals *values.Values, mode string) (*rebaseActionExecutionResult, error) {
	settings_ := &RebaseActionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrapf(err, "failed to decode rebase %s settings", mode)
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect current workspace")
	}
	workflow := workflows.NewRebaseWorkflow(workspace)

	var rows []workflows.RebaseActionRow
	switch mode {
	case "continue":
		rows, err = workflow.Continue(ctx, settings_.Repository, settings_.Jobs)
	case "abort":
		rows, err = workflow.Abort(ctx, settings_.Repository, settings_.Jobs)
	default:
		return nil, errors.Errorf("unsupported rebase action mode: %s", mode)
	}
	if err != nil {
		return nil, err
	}

	return &rebaseActionExecutionResult{
		WorkspaceName: workspace.Name,
		Mode:          mode,
		Jobs:          settings_.Jobs,
		Rows:          rows,
	}, nil
}

func rebaseActionResultToRows(result *rebaseActionExecutionResult) []types.Row {
	rows := make([]types.Row, 0, len(result.Rows))
	for _, row := range result.Rows {
		rows = append(rows, types.NewRow(
			types.MRP("workspace", result.WorkspaceName),
			types.MRP("mode", result.Mode),
			types.MRP("repository", row.Repository),
			types.MRP("success", row.Success),
			types.MRP("error", row.Error),
			types.MRP("jobs", result.Jobs),
		))
	}
	return rows
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

	rebaseCobra, err := wsmcmdcommon.BuildCobraCommandDualMode(rebaseCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase cobra command")
	}
	statusCobra, err := wsmcmdcommon.BuildCobraCommandDualMode(statusCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase status cobra command")
	}
	continueCobra, err := wsmcmdcommon.BuildCobraCommandDualMode(continueCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase continue cobra command")
	}
	abortCobra, err := wsmcmdcommon.BuildCobraCommandDualMode(abortCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build rebase abort cobra command")
	}

	rebaseCobra.AddCommand(statusCobra, continueCobra, abortCobra)
	return rebaseCobra, nil
}
