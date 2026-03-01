package workspace

import (
	"context"
	"fmt"
	"strings"

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
var _ cmds.GlazeCommand = &InfoCommand{}

type infoExecutionResult struct {
	Workspace *wsm.Workspace
	Field     string
	Value     string
	HasField  bool
}

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
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	if result.HasField {
		fmt.Println(result.Value)
		return nil
	}

	if err := printInfoHuman(result.Workspace); err != nil {
		return err
	}

	return nil
}

func (c *InfoCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	row := infoResultToRow(result)
	return gp.AddRow(ctx, row)
}

func (c *InfoCommand) execute(_ context.Context, vals *values.Values) (*infoExecutionResult, error) {
	settings_ := &InfoSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode info settings")
	}

	workflow := workflows.NewInfoWorkflow()
	workspaceName := settings_.WorkspaceName
	if settings_.WorkspaceNameArg != "" {
		workspaceName = settings_.WorkspaceNameArg
	}

	workspace, err := workflow.ResolveWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}

	if settings_.Field != "" {
		value, err := workflow.FieldValue(workspace, settings_.Field)
		if err != nil {
			return nil, err
		}

		return &infoExecutionResult{
			Workspace: workspace,
			Field:     strings.ToLower(settings_.Field),
			Value:     value,
			HasField:  true,
		}, nil
	}

	return &infoExecutionResult{
		Workspace: workspace,
	}, nil
}

func infoResultToRow(result *infoExecutionResult) types.Row {
	if result.HasField {
		return types.NewRow(
			types.MRP("workspace", result.Workspace.Name),
			types.MRP("field", result.Field),
			types.MRP("value", result.Value),
		)
	}

	repositories := make([]string, 0, len(result.Workspace.Repositories))
	for _, repo := range result.Workspace.Repositories {
		repositories = append(repositories, repo.Name)
	}

	return types.NewRow(
		types.MRP("name", result.Workspace.Name),
		types.MRP("path", result.Workspace.Path),
		types.MRP("branch", result.Workspace.Branch),
		types.MRP("repository_count", len(result.Workspace.Repositories)),
		types.MRP("repositories", repositories),
		types.MRP("created", result.Workspace.Created),
		types.MRP("go_workspace", result.Workspace.GoWorkspace),
	)
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
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
