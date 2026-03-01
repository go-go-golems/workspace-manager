package workspace

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
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

// DeleteCommand deletes a workspace.
type DeleteCommand struct {
	*cmds.CommandDescription
}

// DeleteSettings stores parsed delete command settings.
type DeleteSettings struct {
	WorkspaceNameArg string `glazed:"workspace-name"`
	WorkspaceName    string `glazed:"workspace"`
	Force            bool   `glazed:"force"`
	ForceWorktrees   bool   `glazed:"force-worktrees"`
	RemoveFiles      bool   `glazed:"remove-files"`
}

var _ cmds.BareCommand = &DeleteCommand{}
var _ cmds.GlazeCommand = &DeleteCommand{}

type deleteExecutionResult struct {
	WorkspaceName   string
	WorkspacePath   string
	RepositoryCount int
	Force           bool
	ForceWorktrees  bool
	RemoveFiles     bool
	StatusError     string
	Cancelled       bool
}

func NewDeleteCommand() (*DeleteCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"delete",
		cmds.WithShort("Delete a workspace"),
		cmds.WithLong(`Delete a workspace and optionally remove its files.
Use with caution.`),
		cmds.WithFlags(
			fields.New("workspace-name", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Workspace name")),
			fields.New("workspace", fields.TypeString, fields.WithHelp("Workspace name")),
			fields.New("force", fields.TypeBool, fields.WithShortFlag("f"), fields.WithDefault(false), fields.WithHelp("Force delete without confirmation")),
			fields.New("force-worktrees", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Force worktree removal even with uncommitted changes")),
			fields.New("remove-files", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Remove workspace files and directories")),
		),
	)
	if err != nil {
		return nil, err
	}
	return &DeleteCommand{CommandDescription: desc}, nil
}

func (c *DeleteCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals, true, true)
	if err != nil {
		return err
	}

	if result.Cancelled {
		output.PrintInfo("Operation cancelled.")
		return nil
	}

	if result.RemoveFiles {
		output.PrintSuccess("Workspace '%s' and all files deleted successfully", result.WorkspaceName)
	} else {
		output.PrintSuccess("Workspace configuration '%s' deleted successfully", result.WorkspaceName)
		output.PrintInfo("Files remain at: %s", result.WorkspacePath)
	}

	return nil
}

func (c *DeleteCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals, false, false)
	if err != nil {
		return err
	}

	if result.Cancelled {
		return nil
	}

	row := deleteResultToRow(result)
	return gp.AddRow(ctx, row)
}

func (c *DeleteCommand) execute(
	ctx context.Context,
	vals *values.Values,
	emitHuman bool,
	allowPrompt bool,
) (*deleteExecutionResult, error) {
	settings_ := &DeleteSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode delete settings")
	}

	workspaceName := settings_.WorkspaceName
	if settings_.WorkspaceNameArg != "" {
		workspaceName = settings_.WorkspaceNameArg
	}
	if workspaceName == "" {
		return nil, errors.New("workspace name is required")
	}

	workflow, err := workflows.NewDeleteWorkflow()
	if err != nil {
		return nil, err
	}

	preview, err := workflow.Preview(ctx, workspaceName)
	if err != nil {
		return nil, err
	}
	workspace := preview.Workspace

	if emitHuman {
		output.PrintHeader("Current workspace status")
		if preview.StatusError == nil {
			if err := printStatusDetailed(preview.Status, false); err != nil {
				output.PrintError("Error showing status: %v", err)
			}
		} else {
			output.PrintError("Error getting status: %v", preview.StatusError)
		}
		fmt.Printf("\n")

		output.PrintHeader("Workspace: %s", workspace.Name)
		fmt.Printf("  Path: %s\n", workspace.Path)
		fmt.Printf("  Repositories: %d\n", len(workspace.Repositories))

		output.PrintWarning("This will:")
		if settings_.ForceWorktrees {
			fmt.Printf("  1. Remove git worktrees (git worktree remove --force)\n")
		} else {
			fmt.Printf("  1. Remove git worktrees (git worktree remove)\n")
			output.PrintWarning("     Will fail if there are uncommitted changes")
		}
		if settings_.RemoveFiles {
			output.PrintError("  2. DELETE the workspace directory and ALL its contents!")
		} else {
			fmt.Printf("  2. Remove workspace configuration\n")
			fmt.Printf("  3. Clean up workspace-specific files (go.work, AGENT.md)\n")
			fmt.Printf("  4. Repository worktrees will remain at: %s\n", workspace.Path)
		}
	}

	if !settings_.Force {
		if !allowPrompt {
			return nil, errors.New("--force is required when using --with-glaze-output")
		}

		var confirmed bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Are you sure you want to delete workspace '%s'?", workspaceName)).
					Description("This action cannot be undone.").
					Value(&confirmed),
			),
		)
		if err := form.Run(); err != nil {
			if isUserCancelledError(err) {
				return &deleteExecutionResult{Cancelled: true}, nil
			}
			return nil, errors.Wrap(err, "confirmation failed")
		}
		if !confirmed {
			return &deleteExecutionResult{Cancelled: true}, nil
		}
	}

	if err := workflow.Delete(ctx, workspaceName, settings_.RemoveFiles, settings_.ForceWorktrees); err != nil {
		return nil, err
	}

	statusErr := ""
	if preview.StatusError != nil {
		statusErr = preview.StatusError.Error()
	}

	return &deleteExecutionResult{
		WorkspaceName:   workspace.Name,
		WorkspacePath:   workspace.Path,
		RepositoryCount: len(workspace.Repositories),
		Force:           settings_.Force,
		ForceWorktrees:  settings_.ForceWorktrees,
		RemoveFiles:     settings_.RemoveFiles,
		StatusError:     statusErr,
	}, nil
}

func deleteResultToRow(result *deleteExecutionResult) types.Row {
	return types.NewRow(
		types.MRP("workspace", result.WorkspaceName),
		types.MRP("workspace_path", result.WorkspacePath),
		types.MRP("repository_count", result.RepositoryCount),
		types.MRP("force", result.Force),
		types.MRP("force_worktrees", result.ForceWorktrees),
		types.MRP("remove_files", result.RemoveFiles),
		types.MRP("status_error", result.StatusError),
		types.MRP("status", "deleted"),
	)
}

func NewDeleteCobraCommand() (*cobra.Command, error) {
	command, err := NewDeleteCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build delete command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
