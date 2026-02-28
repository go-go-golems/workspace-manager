package workspace

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/types"
	wsmcmdcommon "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/common"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// MergeCommand merges a forked workspace back to its base branch.
type MergeCommand struct {
	*cmds.CommandDescription
}

// MergeSettings stores parsed merge command settings.
type MergeSettings struct {
	WorkspaceNameArg string `glazed:"workspace-name"`
	WorkspaceName    string `glazed:"workspace"`
	DryRun           bool   `glazed:"dry-run"`
	Force            bool   `glazed:"force"`
	KeepWorkspace    bool   `glazed:"keep-workspace"`
}

var _ cmds.BareCommand = &MergeCommand{}

func NewMergeCommand() (*MergeCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"merge",
		cmds.WithShort("Merge a forked workspace back into its base branch"),
		cmds.WithLong(`Merge a forked workspace back into its base branch and optionally delete the workspace.`),
		cmds.WithFlags(
			fields.New("workspace-name", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Workspace name")),
			fields.New("workspace", fields.TypeString, fields.WithHelp("Workspace name")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Show what would be merged without executing")),
			fields.New("force", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Skip confirmation prompts")),
			fields.New("keep-workspace", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Keep the workspace after merge")),
		),
	)
	if err != nil {
		return nil, err
	}
	return &MergeCommand{CommandDescription: desc}, nil
}

func (c *MergeCommand) Run(ctx context.Context, vals *values.Values) error {
	settings_ := &MergeSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode merge settings")
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	workspaceName := settings_.WorkspaceName
	if settings_.WorkspaceNameArg != "" {
		workspaceName = settings_.WorkspaceNameArg
	}

	workflow := workflows.NewMergeWorkflow()
	if err := workflow.Execute(ctx, workflows.MergeRequest{
		WorkspaceName: workspaceName,
		DryRun:        settings_.DryRun,
		Force:         settings_.Force,
		KeepWorkspace: settings_.KeepWorkspace,
	}); err != nil {
		return err
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		rows := []types.Row{types.NewRow(
			types.MRP("workspace", workspaceName),
			types.MRP("dry_run", settings_.DryRun),
			types.MRP("force", settings_.Force),
			types.MRP("keep_workspace", settings_.KeepWorkspace),
			types.MRP("status", "completed"),
		)}
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit merge rows")
		}
	}

	return nil
}

func NewMergeCobraCommand() (*cobra.Command, error) {
	command, err := NewMergeCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build merge command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommand(command)
}
