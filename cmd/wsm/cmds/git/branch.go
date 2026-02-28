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

var _ cmds.BareCommand = &BranchCreateCommand{}
var _ cmds.BareCommand = &BranchSwitchCommand{}
var _ cmds.BareCommand = &BranchListCommand{}

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
	settings_ := &BranchCreateSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode branch create settings")
	}
	if settings_.BranchName == "" {
		return errors.New("branch name is required")
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	branchOps := wsm.NewBranchOperations(workspace)
	results, err := branchOps.CreateBranch(ctx, settings_.BranchName, settings_.Track)
	if err != nil {
		return errors.Wrap(err, "branch creation failed")
	}

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		output.PrintHeader("Creating branch '%s' across workspace: %s", settings_.BranchName, workspace.Name)
		if err := printBranchResults(results, "create"); err != nil {
			return err
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		rows := make([]types.Row, 0, len(results))
		for _, result := range results {
			rows = append(rows, types.NewRow(
				types.MRP("workspace", workspace.Name),
				types.MRP("operation", "create"),
				types.MRP("branch", settings_.BranchName),
				types.MRP("track", settings_.Track),
				types.MRP("repository", result.Repository),
				types.MRP("success", result.Success),
				types.MRP("error", result.Error),
			))
		}
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit branch create rows")
		}
	}

	return nil
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
	settings_ := &BranchSwitchSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode branch switch settings")
	}
	if settings_.BranchName == "" {
		return errors.New("branch name is required")
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	branchOps := wsm.NewBranchOperations(workspace)
	results, err := branchOps.SwitchBranch(ctx, settings_.BranchName)
	if err != nil {
		return errors.Wrap(err, "branch switch failed")
	}

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		output.PrintHeader("Switching to branch '%s' across workspace: %s", settings_.BranchName, workspace.Name)
		if err := printBranchResults(results, "switch"); err != nil {
			return err
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		rows := make([]types.Row, 0, len(results))
		for _, result := range results {
			rows = append(rows, types.NewRow(
				types.MRP("workspace", workspace.Name),
				types.MRP("operation", "switch"),
				types.MRP("branch", settings_.BranchName),
				types.MRP("repository", result.Repository),
				types.MRP("success", result.Success),
				types.MRP("error", result.Error),
			))
		}
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit branch switch rows")
		}
	}

	return nil
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
	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	checker := wsm.NewStatusChecker()
	rows := make([]types.Row, 0, len(workspace.Repositories))

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		output.PrintHeader("Current branches in workspace: %s", workspace.Name)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if wsmcmdcommon.ShouldOutputHuman(mode) {
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
	}

	for _, repo := range workspace.Repositories {
		status, err := checker.GetWorkspaceStatus(ctx, &wsm.Workspace{
			Path:         workspace.Path,
			Repositories: []wsm.Repository{repo},
		})

		branchName := "unknown"
		symbol := "❌"
		errMsg := ""
		hasChanges := false
		hasConflicts := false

		if err == nil && len(status.Repositories) > 0 {
			repoStatus := status.Repositories[0]
			branchName = repoStatus.CurrentBranch
			symbol = "✅"
			if repoStatus.HasChanges {
				symbol = "🔄"
				hasChanges = true
			}
			if repoStatus.HasConflicts {
				symbol = "⚠️"
				hasConflicts = true
			}
		} else if err != nil {
			errMsg = err.Error()
		}

		if wsmcmdcommon.ShouldOutputHuman(mode) {
			fmt.Fprintf(w, "%s\t%s\t%s\n", repo.Name, branchName, symbol)
		}

		rows = append(rows, types.NewRow(
			types.MRP("workspace", workspace.Name),
			types.MRP("repository", repo.Name),
			types.MRP("current_branch", branchName),
			types.MRP("status_symbol", symbol),
			types.MRP("has_changes", hasChanges),
			types.MRP("has_conflicts", hasConflicts),
			types.MRP("error", errMsg),
		))
	}

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		fmt.Fprintln(w)
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit branch list rows")
		}
	}

	return nil
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

	createCobra, err := wsmcmdcommon.BuildCobraCommand(createCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build branch create cobra command")
	}
	switchCobra, err := wsmcmdcommon.BuildCobraCommand(switchCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build branch switch cobra command")
	}
	listCobra, err := wsmcmdcommon.BuildCobraCommand(listCmd)
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
