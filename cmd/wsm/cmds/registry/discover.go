package registry

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

func NewDiscoverCommand() (*DiscoverCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"discover",
		cmds.WithShort("Discover git repositories in specified directories"),
		cmds.WithLong(`Discover git repositories in the specified directories and add them to the registry.
If no paths are specified, defaults to current directory.

Output behavior:
  --output-mode human   Human-readable output (default)
  --output-mode data    Structured glazed output only
  --output-mode both    Human output first, then structured output`),
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
	settings_ := &DiscoverSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode discover settings")
	}

	workflow, err := workflows.NewDiscoverWorkflow()
	if err != nil {
		return err
	}

	result, err := workflow.Discover(ctx, workflows.DiscoverRequest{
		Paths:     settings_.Paths,
		Recursive: settings_.Recursive,
		MaxDepth:  settings_.MaxDepth,
	})
	if err != nil {
		return err
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if wsmcmdcommon.ShouldOutputHuman(mode) {
		output.PrintInfo("Discovering repositories in %v", result.Paths)
		output.PrintSuccess("Discovery complete! Found %d repositories", result.RepositoryCount)
		if result.RepositoryCount > 0 {
			output.PrintInfo("Use 'wsm list repos' to see all discovered repositories")
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		rows := []types.Row{types.NewRow(
			types.MRP("paths", result.Paths),
			types.MRP("repository_count", result.RepositoryCount),
		)}
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit discover rows")
		}
	}

	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	return nil
}

func NewDiscoverCobraCommand() (*cobra.Command, error) {
	command, err := NewDiscoverCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build discover command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommand(command)
}
