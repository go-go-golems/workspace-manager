package workspace

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
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type PathCommand struct {
	*cmds.CommandDescription
}

type PathSettings struct {
	WorkspaceNameArg string `glazed:"workspace-name"`
	WorkspaceName    string `glazed:"workspace"`
}

var _ cmds.BareCommand = &PathCommand{}
var _ cmds.GlazeCommand = &PathCommand{}

type pathExecutionResult struct {
	Workspace *wsm.Workspace
}

func NewPathCommand() (*PathCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"path",
		cmds.WithShort("Get workspace filesystem path"),
		cmds.WithLong(`Output the full filesystem path for a workspace.

Useful for shell integration. If no workspace name is provided, attempts to
detect the current workspace from the working directory.

Examples:
  # Get path of a specific workspace
  wsm path my-workspace

  # Use with cd
  cd $(wsm path my-workspace)

  # Detect current workspace path
  wsm path`),
		cmds.WithFlags(
			fields.New(
				"workspace-name",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Workspace name"),
			),
			fields.New(
				"workspace",
				fields.TypeString,
				fields.WithHelp("Workspace name"),
			),
		),
	)
	if err != nil {
		return nil, err
	}

	return &PathCommand{CommandDescription: desc}, nil
}

func (c *PathCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}
	fmt.Println(result.Workspace.Path)
	return nil
}

func (c *PathCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}
	row := types.NewRow(
		types.MRP("workspace", result.Workspace.Name),
		types.MRP("path", result.Workspace.Path),
	)
	return gp.AddRow(ctx, row)
}

func (c *PathCommand) execute(ctx context.Context, vals *values.Values) (*pathExecutionResult, error) {
	settings := &PathSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return nil, errors.Wrap(err, "failed to decode path settings")
	}

	workspaceName := settings.WorkspaceName
	if settings.WorkspaceNameArg != "" {
		workspaceName = settings.WorkspaceNameArg
	}

	workflow := workflows.NewInfoWorkflow()
	workspace, err := workflow.ResolveWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}

	return &pathExecutionResult{Workspace: workspace}, nil
}

func NewPathCobraCommand() (*cobra.Command, error) {
	command, err := NewPathCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build path command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
