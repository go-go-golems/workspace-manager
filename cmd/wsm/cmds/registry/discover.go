package registry

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
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DiscoverCommand discovers repositories and supports dual output modes.
type DiscoverCommand struct {
	*cmds.CommandDescription
}

// DiscoverSettings stores parsed discover command settings.
type DiscoverSettings struct {
	Paths     []string `glazed:"paths"`
	Recursive bool     `glazed:"recursive"`
	MaxDepth  int      `glazed:"max-depth"`
}

var _ cmds.BareCommand = &DiscoverCommand{}
var _ cmds.GlazeCommand = &DiscoverCommand{}

type discoverExecutionResult struct {
	Paths           []string
	RepositoryCount int
}

func NewDiscoverCommand() (*DiscoverCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"discover",
		cmds.WithShort("Discover git repositories in specified directories"),
		cmds.WithLong(`Discover git repositories in the specified directories and add them to the registry.
If no paths are specified, defaults to current directory.

Use --with-glaze-output to emit structured rows.`),
		cmds.WithFlags(
			fields.New(
				"paths",
				fields.TypeStringList,
				fields.WithIsArgument(true),
				fields.WithHelp("Directories to scan for repositories"),
			),
			fields.New(
				"recursive",
				fields.TypeBool,
				fields.WithShortFlag("r"),
				fields.WithDefault(true),
				fields.WithHelp("Recursively scan subdirectories"),
			),
			fields.New(
				"max-depth",
				fields.TypeInteger,
				fields.WithDefault(3),
				fields.WithHelp("Maximum depth for recursive scanning"),
			),
		),
	)
	if err != nil {
		return nil, err
	}

	return &DiscoverCommand{CommandDescription: desc}, nil
}

func (c *DiscoverCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	output.PrintInfo("Discovering repositories in %v", result.Paths)
	output.PrintSuccess("Discovery complete! Found %d repositories", result.RepositoryCount)
	if result.RepositoryCount > 0 {
		output.PrintInfo("Use 'wsm list repos' to see all discovered repositories")
	}

	return nil
}

func (c *DiscoverCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	row := types.NewRow(
		types.MRP("paths", result.Paths),
		types.MRP("repository_count", result.RepositoryCount),
	)
	return gp.AddRow(ctx, row)
}

func (c *DiscoverCommand) execute(ctx context.Context, vals *values.Values) (*discoverExecutionResult, error) {
	settings_ := &DiscoverSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode discover settings")
	}

	workflow, err := workflows.NewDiscoverWorkflow()
	if err != nil {
		return nil, err
	}

	result, err := workflow.Discover(ctx, workflows.DiscoverRequest{
		Paths:     settings_.Paths,
		Recursive: settings_.Recursive,
		MaxDepth:  settings_.MaxDepth,
	})
	if err != nil {
		return nil, err
	}

	return &discoverExecutionResult{
		Paths:           result.Paths,
		RepositoryCount: result.RepositoryCount,
	}, nil
}

func NewDiscoverCobraCommand() (*cobra.Command, error) {
	command, err := NewDiscoverCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build discover command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
