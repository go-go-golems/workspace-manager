package git

import (
	"context"
	"fmt"

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

// CommitCommand commits changes across workspace repositories.
type CommitCommand struct {
	*cmds.CommandDescription
}

// CommitSettings stores parsed commit command settings.
type CommitSettings struct {
	Message        string `glazed:"message"`
	Interactive    bool   `glazed:"interactive"`
	AddAll         bool   `glazed:"add-all"`
	Push           bool   `glazed:"push"`
	DryRun         bool   `glazed:"dry-run"`
	CommitTemplate string `glazed:"commit-template"`
}

var _ cmds.BareCommand = &CommitCommand{}
var _ cmds.GlazeCommand = &CommitCommand{}

type commitExecutionResult struct {
	WorkspaceName   string
	Message         string
	Interactive     bool
	AddAll          bool
	Push            bool
	DryRun          bool
	SelectedChanges map[string][]wsm.FileChange
	Status          string
}

func NewCommitCommand() (*CommitCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"commit",
		cmds.WithShort("Commit changes across workspace repositories"),
		cmds.WithLong(`Commit related changes across multiple repositories in the workspace.
Supports interactive file selection and consistent commit messaging.`),
		cmds.WithFlags(
			fields.New("message", fields.TypeString, fields.WithShortFlag("m"), fields.WithHelp("Commit message")),
			fields.New("interactive", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Interactive file selection")),
			fields.New("add-all", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Add all changes")),
			fields.New("push", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Push changes after commit")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Show what would be committed")),
			fields.New("commit-template", fields.TypeString, fields.WithHelp("Use commit message template")),
		),
	)
	if err != nil {
		return nil, err
	}
	return &CommitCommand{CommandDescription: desc}, nil
}

func (c *CommitCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals, true)
	if err != nil {
		return err
	}

	switch result.Status {
	case "no_changes":
		output.PrintInfo("No changes found in workspace")
	case "no_selection":
		output.PrintInfo("No files selected for commit")
	case "executed":
		if !result.DryRun {
			output.PrintSuccess("Successfully committed changes across %d repositories", len(result.SelectedChanges))
			if result.Push {
				output.PrintInfo("Changes pushed to remote repositories")
			}
		}
	}

	return nil
}

func (c *CommitCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals, false)
	if err != nil {
		return err
	}

	for _, row := range commitResultToRows(result) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add commit row")
		}
	}

	return nil
}

func (c *CommitCommand) execute(ctx context.Context, vals *values.Values, allowInteractive bool) (*commitExecutionResult, error) {
	settings_ := &CommitSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode commit settings")
	}

	if settings_.Interactive && !allowInteractive {
		return nil, errors.New("--interactive is only supported in human output mode")
	}

	workflow := workflows.NewCommitWorkflow()
	req := workflows.CommitRequest{
		Message:  settings_.Message,
		Template: settings_.CommitTemplate,
		AddAll:   settings_.AddAll,
		Push:     settings_.Push,
		DryRun:   settings_.DryRun,
	}

	prep, err := workflow.Prepare(ctx, req)
	if err != nil {
		return nil, err
	}

	allChanges := prep.Changes
	if len(allChanges) == 0 {
		return &commitExecutionResult{
			WorkspaceName: prep.Workspace.Name,
			Message:       prep.Message,
			Interactive:   settings_.Interactive,
			AddAll:        settings_.AddAll,
			Push:          settings_.Push,
			DryRun:        settings_.DryRun,
			Status:        "no_changes",
		}, nil
	}

	if prep.Message == "" && !settings_.Interactive {
		return nil, errors.New("commit message is required. Use -m flag or --interactive mode")
	}

	selectedChanges := allChanges
	if settings_.Interactive {
		selectedChanges, prep.Message, err = selectChangesInteractively(allChanges, prep.Message)
		if err != nil {
			return nil, errors.Wrap(err, "interactive selection failed")
		}
	}

	if len(selectedChanges) == 0 {
		return &commitExecutionResult{
			WorkspaceName: prep.Workspace.Name,
			Message:       prep.Message,
			Interactive:   settings_.Interactive,
			AddAll:        settings_.AddAll,
			Push:          settings_.Push,
			DryRun:        settings_.DryRun,
			Status:        "no_selection",
		}, nil
	}

	if err := workflow.Execute(ctx, prep, selectedChanges, req); err != nil {
		return nil, err
	}

	return &commitExecutionResult{
		WorkspaceName:   prep.Workspace.Name,
		Message:         prep.Message,
		Interactive:     settings_.Interactive,
		AddAll:          settings_.AddAll,
		Push:            settings_.Push,
		DryRun:          settings_.DryRun,
		SelectedChanges: selectedChanges,
		Status:          "executed",
	}, nil
}

func commitResultToRows(result *commitExecutionResult) []types.Row {
	fileCount := 0
	for _, files := range result.SelectedChanges {
		fileCount += len(files)
	}

	rows := []types.Row{types.NewRow(
		types.MRP("workspace", result.WorkspaceName),
		types.MRP("message", result.Message),
		types.MRP("interactive", result.Interactive),
		types.MRP("add_all", result.AddAll),
		types.MRP("push", result.Push),
		types.MRP("dry_run", result.DryRun),
		types.MRP("selected_repositories", len(result.SelectedChanges)),
		types.MRP("selected_files", fileCount),
		types.MRP("status", result.Status),
	)}

	if result.Status != "executed" {
		return rows
	}

	for repoName, files := range result.SelectedChanges {
		rows = append(rows, types.NewRow(
			types.MRP("workspace", result.WorkspaceName),
			types.MRP("repository", repoName),
			types.MRP("file_count", len(files)),
			types.MRP("files", files),
		))
	}

	return rows
}

func selectChangesInteractively(allChanges map[string][]wsm.FileChange, initialMessage string) (map[string][]wsm.FileChange, string, error) {
	output.PrintHeader("Interactive Commit")
	fmt.Println()

	output.PrintInfo("Changes found:")
	repoIndex := 0
	for repoName, changes := range allChanges {
		fmt.Printf("\n%d. Repository: %s (%d files)\n", repoIndex+1, repoName, len(changes))
		for i, change := range changes {
			status := wsm.GetStatusSymbol(change.Status)
			staged := ""
			if change.Staged {
				staged = " (staged)"
			}
			fmt.Printf("   %c. %s %s%s\n", 'a'+i, status, change.FilePath, staged)
		}
		repoIndex++
	}
	fmt.Println()

	message := initialMessage
	if message == "" {
		fmt.Print("Commit message: ")
		if _, err := fmt.Scanln(&message); err != nil {
			return nil, "", errors.Wrap(err, "failed to read commit message")
		}
		if message == "" {
			return nil, "", errors.New("commit message is required")
		}
	}

	output.PrintInfo("Proceeding with all changes...")
	return allChanges, message, nil
}

func NewCommitCobraCommand() (*cobra.Command, error) {
	command, err := NewCommitCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build commit command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
