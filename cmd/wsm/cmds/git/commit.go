package git

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
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
	settings_ := &CommitSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode commit settings")
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
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
		return err
	}

	allChanges := prep.Changes
	if len(allChanges) == 0 {
		if wsmcmdcommon.ShouldOutputHuman(mode) {
			output.PrintInfo("No changes found in workspace")
		}
		if wsmcmdcommon.ShouldOutputData(mode) {
			rows := []types.Row{types.NewRow(
				types.MRP("workspace", prep.Workspace.Name),
				types.MRP("message", prep.Message),
				types.MRP("interactive", settings_.Interactive),
				types.MRP("add_all", settings_.AddAll),
				types.MRP("push", settings_.Push),
				types.MRP("dry_run", settings_.DryRun),
				types.MRP("selected_repositories", 0),
				types.MRP("selected_files", 0),
				types.MRP("status", "no_changes"),
			)}
			if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
				return errors.Wrap(err, "failed to emit commit rows")
			}
		}
		return nil
	}

	if prep.Message == "" && !settings_.Interactive {
		return errors.New("commit message is required. Use -m flag or --interactive mode")
	}

	selectedChanges := allChanges
	if settings_.Interactive {
		selectedChanges, prep.Message, err = selectChangesInteractively(allChanges, prep.Message)
		if err != nil {
			return errors.Wrap(err, "interactive selection failed")
		}
	}

	if len(selectedChanges) == 0 {
		if wsmcmdcommon.ShouldOutputHuman(mode) {
			output.PrintInfo("No files selected for commit")
		}
		if wsmcmdcommon.ShouldOutputData(mode) {
			rows := []types.Row{types.NewRow(
				types.MRP("workspace", prep.Workspace.Name),
				types.MRP("message", prep.Message),
				types.MRP("interactive", settings_.Interactive),
				types.MRP("add_all", settings_.AddAll),
				types.MRP("push", settings_.Push),
				types.MRP("dry_run", settings_.DryRun),
				types.MRP("selected_repositories", 0),
				types.MRP("selected_files", 0),
				types.MRP("status", "no_selection"),
			)}
			if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
				return errors.Wrap(err, "failed to emit commit rows")
			}
		}
		return nil
	}

	if err := workflow.Execute(ctx, prep, selectedChanges, req); err != nil {
		return err
	}

	if wsmcmdcommon.ShouldOutputHuman(mode) && !settings_.DryRun {
		output.PrintSuccess("Successfully committed changes across %d repositories", len(selectedChanges))
		if settings_.Push {
			output.PrintInfo("Changes pushed to remote repositories")
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		fileCount := 0
		for _, files := range selectedChanges {
			fileCount += len(files)
		}
		rows := make([]types.Row, 0, len(selectedChanges)+1)
		rows = append(rows, types.NewRow(
			types.MRP("workspace", prep.Workspace.Name),
			types.MRP("message", prep.Message),
			types.MRP("interactive", settings_.Interactive),
			types.MRP("add_all", settings_.AddAll),
			types.MRP("push", settings_.Push),
			types.MRP("dry_run", settings_.DryRun),
			types.MRP("selected_repositories", len(selectedChanges)),
			types.MRP("selected_files", fileCount),
			types.MRP("status", "executed"),
		))
		for repoName, files := range selectedChanges {
			rows = append(rows, types.NewRow(
				types.MRP("workspace", prep.Workspace.Name),
				types.MRP("repository", repoName),
				types.MRP("file_count", len(files)),
				types.MRP("files", files),
			))
		}
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit commit rows")
		}
	}

	return nil
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
	return wsmcmdcommon.BuildCobraCommand(command)
}
