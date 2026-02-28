package workspace

import (
	"context"
	"fmt"
	"strings"

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

// InfoCommand displays workspace information.
type InfoCommand struct {
	*cmds.CommandDescription
}

// InfoSettings stores parsed info command settings.
type InfoSettings struct {
	WorkspaceNameArg string `glazed:"workspace-name"`
	WorkspaceName    string `glazed:"workspace"`
	Field            string `glazed:"field"`
}

var _ cmds.BareCommand = &InfoCommand{}

func NewInfoCommand() (*InfoCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"info",
		cmds.WithShort("Display workspace information"),
		cmds.WithLong(`Display information about a workspace.

By default, shows all workspace information. Use --field to get a specific piece of information.

Available fields:
  - path, name, branch, repositories, created, date, time`),
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
				"field",
				fields.TypeString,
				fields.WithHelp("Output specific field only (path, name, branch, repositories, created, date, time)"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &InfoCommand{CommandDescription: desc}, nil
}

func (c *InfoCommand) Run(ctx context.Context, vals *values.Values) error {
	settings_ := &InfoSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode info settings")
	}

	workflow := workflows.NewInfoWorkflow()
	workspaceName := settings_.WorkspaceName
	if settings_.WorkspaceNameArg != "" {
		workspaceName = settings_.WorkspaceNameArg
	}

	workspace, err := workflow.ResolveWorkspace(workspaceName)
	if err != nil {
		return err
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)

	if settings_.Field != "" {
		value, err := workflow.FieldValue(workspace, settings_.Field)
		if err != nil {
			return err
		}

		if wsmcmdcommon.ShouldOutputHuman(mode) {
			fmt.Println(value)
		}
		if wsmcmdcommon.ShouldOutputData(mode) {
			rows := []types.Row{types.NewRow(
				types.MRP("workspace", workspace.Name),
				types.MRP("field", strings.ToLower(settings_.Field)),
				types.MRP("value", value),
			)}
			if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
				return errors.Wrap(err, "failed to emit info field row")
			}
		}
		if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
			return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
		}
		return nil
	}

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		if err := printInfoHuman(workspace); err != nil {
			return err
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		repositories := make([]string, 0, len(workspace.Repositories))
		for _, repo := range workspace.Repositories {
			repositories = append(repositories, repo.Name)
		}

		rows := []types.Row{types.NewRow(
			types.MRP("name", workspace.Name),
			types.MRP("path", workspace.Path),
			types.MRP("branch", workspace.Branch),
			types.MRP("repository_count", len(workspace.Repositories)),
			types.MRP("repositories", repositories),
			types.MRP("created", workspace.Created),
			types.MRP("go_workspace", workspace.GoWorkspace),
		)}
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit info rows")
		}
	}

	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	return nil
}

func printInfoHuman(workspace *wsm.Workspace) error {
	output.PrintHeader("Workspace Information")
	fmt.Printf("  Name:         %s\n", workspace.Name)
	fmt.Printf("  Path:         %s\n", workspace.Path)
	fmt.Printf("  Branch:       %s\n", workspace.Branch)
	fmt.Printf("  Repositories: %d\n", len(workspace.Repositories))
	fmt.Printf("  Created:      %s\n", workspace.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Go Workspace: %t\n", workspace.GoWorkspace)

	if len(workspace.Repositories) > 0 {
		output.PrintHeader("\nRepositories")
		for _, repo := range workspace.Repositories {
			fmt.Printf("  - %s (%s)\n", repo.Name, repo.RemoteURL)
		}
	}

	return nil
}

func NewInfoCobraCommand() (*cobra.Command, error) {
	command, err := NewInfoCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build info command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommand(command)
}
