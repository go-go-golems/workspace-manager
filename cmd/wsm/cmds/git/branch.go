package git

import (
	"context"
	"fmt"
	"os"
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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// BranchCreateCommand creates a branch across all repositories.
type BranchCreateCommand struct {
	*cmds.CommandDescription
}

// BranchSwitchCommand switches branch across all repositories.
type BranchSwitchCommand struct {
	*cmds.CommandDescription
}

// BranchListCommand lists current branches across repositories.
type BranchListCommand struct {
	*cmds.CommandDescription
}

type BranchCreateSettings struct {
	BranchName string `glazed:"branch-name"`
	Track      bool   `glazed:"track"`
}

type BranchSwitchSettings struct {
	BranchName string `glazed:"branch-name"`
}

type BranchListSettings struct{}

type branchCreateExecutionResult struct {
	WorkspaceName string
	BranchName    string
	Track         bool
	Results       []wsm.BranchOperationResult
}

type branchSwitchExecutionResult struct {
	WorkspaceName string
	BranchName    string
	Results       []wsm.BranchOperationResult
}

type branchListEntry struct {
	Repository    string
	CurrentBranch string
	StatusSymbol  string
	HasChanges    bool
	HasConflicts  bool
	Error         string
}

type branchListExecutionResult struct {
	WorkspaceName string
	Entries       []branchListEntry
}

var _ cmds.BareCommand = &BranchCreateCommand{}
var _ cmds.GlazeCommand = &BranchCreateCommand{}
var _ cmds.BareCommand = &BranchSwitchCommand{}
var _ cmds.GlazeCommand = &BranchSwitchCommand{}
var _ cmds.BareCommand = &BranchListCommand{}
var _ cmds.GlazeCommand = &BranchListCommand{}

func NewBranchCreateCommand() (*BranchCreateCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"create",
		cmds.WithShort("Create a branch across all repositories"),
		cmds.WithLong("Create a new branch with the same name across all repositories in the workspace."),
		cmds.WithFlags(
			fields.New(
				"branch-name",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Branch name"),
			),
			fields.New(
				"track",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Set up tracking for the new branch"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &BranchCreateCommand{CommandDescription: desc}, nil
}

func (c *BranchCreateCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	output.PrintHeader("Creating branch '%s' across workspace: %s", result.BranchName, result.WorkspaceName)
	if err := printBranchResults(result.Results, "create"); err != nil {
		return err
	}

	return nil
}

func (c *BranchCreateCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	for _, row := range branchCreateResultToRows(result) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add branch create row")
		}
	}

	return nil
}

func (c *BranchCreateCommand) execute(ctx context.Context, vals *values.Values) (*branchCreateExecutionResult, error) {
	settings_ := &BranchCreateSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode branch create settings")
	}
	if settings_.BranchName == "" {
		return nil, errors.New("branch name is required")
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect current workspace")
	}

	branchOps := wsm.NewBranchOperations(workspace)
	results, err := branchOps.CreateBranch(ctx, settings_.BranchName, settings_.Track)
	if err != nil {
		return nil, errors.Wrap(err, "branch creation failed")
	}

	return &branchCreateExecutionResult{
		WorkspaceName: workspace.Name,
		BranchName:    settings_.BranchName,
		Track:         settings_.Track,
		Results:       results,
	}, nil
}

func branchCreateResultToRows(result *branchCreateExecutionResult) []types.Row {
	rows := make([]types.Row, 0, len(result.Results))
	for _, operationResult := range result.Results {
		rows = append(rows, types.NewRow(
			types.MRP("workspace", result.WorkspaceName),
			types.MRP("operation", "create"),
			types.MRP("branch", result.BranchName),
			types.MRP("track", result.Track),
			types.MRP("repository", operationResult.Repository),
			types.MRP("success", operationResult.Success),
			types.MRP("error", operationResult.Error),
		))
	}
	return rows
}

func NewBranchSwitchCommand() (*BranchSwitchCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"switch",
		cmds.WithShort("Switch to a branch across all repositories"),
		cmds.WithLong("Switch all repositories in the workspace to the specified branch."),
		cmds.WithFlags(
			fields.New(
				"branch-name",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Branch name"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &BranchSwitchCommand{CommandDescription: desc}, nil
}

func (c *BranchSwitchCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	output.PrintHeader("Switching to branch '%s' across workspace: %s", result.BranchName, result.WorkspaceName)
	if err := printBranchResults(result.Results, "switch"); err != nil {
		return err
	}

	return nil
}

func (c *BranchSwitchCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	for _, row := range branchSwitchResultToRows(result) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add branch switch row")
		}
	}

	return nil
}

func (c *BranchSwitchCommand) execute(ctx context.Context, vals *values.Values) (*branchSwitchExecutionResult, error) {
	settings_ := &BranchSwitchSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode branch switch settings")
	}
	if settings_.BranchName == "" {
		return nil, errors.New("branch name is required")
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect current workspace")
	}

	branchOps := wsm.NewBranchOperations(workspace)
	results, err := branchOps.SwitchBranch(ctx, settings_.BranchName)
	if err != nil {
		return nil, errors.Wrap(err, "branch switch failed")
	}

	return &branchSwitchExecutionResult{
		WorkspaceName: workspace.Name,
		BranchName:    settings_.BranchName,
		Results:       results,
	}, nil
}

func branchSwitchResultToRows(result *branchSwitchExecutionResult) []types.Row {
	rows := make([]types.Row, 0, len(result.Results))
	for _, operationResult := range result.Results {
		rows = append(rows, types.NewRow(
			types.MRP("workspace", result.WorkspaceName),
			types.MRP("operation", "switch"),
			types.MRP("branch", result.BranchName),
			types.MRP("repository", operationResult.Repository),
			types.MRP("success", operationResult.Success),
			types.MRP("error", operationResult.Error),
		))
	}
	return rows
}

func NewBranchListCommand() (*BranchListCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"list",
		cmds.WithShort("List current branches across repositories"),
		cmds.WithLong("Show the current branch for each repository in the workspace."),
	)
	if err != nil {
		return nil, err
	}
	return &BranchListCommand{CommandDescription: desc}, nil
}

func (c *BranchListCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	output.PrintHeader("Current branches in workspace: %s", result.WorkspaceName)
	printBranchListHuman(result.Entries)

	return nil
}

func (c *BranchListCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	for _, row := range branchListResultToRows(result) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add branch list row")
		}
	}

	return nil
}

func (c *BranchListCommand) execute(ctx context.Context, _ *values.Values) (*branchListExecutionResult, error) {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect current workspace")
	}

	checker := wsm.NewStatusChecker()
	entries := make([]branchListEntry, 0, len(workspace.Repositories))

	for _, repo := range workspace.Repositories {
		status, err := checker.GetWorkspaceStatus(ctx, &wsm.Workspace{
			Path:         workspace.Path,
			Repositories: []wsm.Repository{repo},
		})

		entry := branchListEntry{
			Repository:    repo.Name,
			CurrentBranch: "unknown",
			StatusSymbol:  "❌",
		}

		if err == nil && len(status.Repositories) > 0 {
			repoStatus := status.Repositories[0]
			entry.CurrentBranch = repoStatus.CurrentBranch
			entry.StatusSymbol = "✅"
			if repoStatus.HasChanges {
				entry.StatusSymbol = "🔄"
				entry.HasChanges = true
			}
			if repoStatus.HasConflicts {
				entry.StatusSymbol = "⚠️"
				entry.HasConflicts = true
			}
		} else if err != nil {
			entry.Error = err.Error()
		}

		entries = append(entries, entry)
	}

	return &branchListExecutionResult{
		WorkspaceName: workspace.Name,
		Entries:       entries,
	}, nil
}

func printBranchListHuman(entries []branchListEntry) {
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

	fmt.Fprintln(w, "\nREPOSITORY\tCURRENT BRANCH\tSTATUS")
	fmt.Fprintln(w, "----------\t--------------\t------")

	for _, entry := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\n", entry.Repository, entry.CurrentBranch, entry.StatusSymbol)
	}
	fmt.Fprintln(w)
}

func branchListResultToRows(result *branchListExecutionResult) []types.Row {
	rows := make([]types.Row, 0, len(result.Entries))
	for _, entry := range result.Entries {
		rows = append(rows, types.NewRow(
			types.MRP("workspace", result.WorkspaceName),
			types.MRP("repository", entry.Repository),
			types.MRP("current_branch", entry.CurrentBranch),
			types.MRP("status_symbol", entry.StatusSymbol),
			types.MRP("has_changes", entry.HasChanges),
			types.MRP("has_conflicts", entry.HasConflicts),
			types.MRP("error", entry.Error),
		))
	}
	return rows
}

func printBranchResults(results []wsm.BranchOperationResult, operation string) error {
	if len(results) == 0 {
		output.PrintInfo("No repositories found.")
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

		fmt.Fprintf(w, "%s\t%s\t%s\n", result.Repository, status, errorMsg)
	}

	fmt.Fprintln(w)
	output.PrintSuccess("Summary: %d/%d repositories %s successfully", successCount, len(results), operation)
	if successCount < len(results) {
		output.PrintWarning("Some repositories failed. Check errors above and resolve manually.")
	}

	return nil
}

func NewBranchCobraCommand() (*cobra.Command, error) {
	createCmd, err := NewBranchCreateCommand()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build branch create command")
	}
	switchCmd, err := NewBranchSwitchCommand()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build branch switch command")
	}
	listCmd, err := NewBranchListCommand()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build branch list command")
	}

	createCobra, err := wsmcmdcommon.BuildCobraCommandDualMode(createCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build branch create cobra command")
	}
	switchCobra, err := wsmcmdcommon.BuildCobraCommandDualMode(switchCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build branch switch cobra command")
	}
	listCobra, err := wsmcmdcommon.BuildCobraCommandDualMode(listCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build branch list cobra command")
	}

	branchCmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage branches across workspace repositories",
		Long: `Create, switch, and manage branches across all repositories in the workspace.
This ensures consistent branch operations across your multi-repository development.`,
	}
	branchCmd.AddCommand(createCobra, switchCobra, listCobra)
	return branchCmd, nil
}
