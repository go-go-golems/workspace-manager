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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DiffCommand shows diffs across workspace repositories.
type DiffCommand struct {
	*cmds.CommandDescription
}

// DiffSettings stores parsed diff command settings.
type DiffSettings struct {
	Staged bool   `glazed:"staged"`
	Repo   string `glazed:"repo"`
	Jobs   int    `glazed:"jobs"`
}

var _ cmds.BareCommand = &DiffCommand{}
var _ cmds.GlazeCommand = &DiffCommand{}

type diffExecutionResult struct {
	Workspace  *wsm.Workspace
	Diff       string
	Staged     bool
	RepoFilter string
	Jobs       int
	HasChanges bool
}

func NewDiffCommand() (*DiffCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"diff",
		cmds.WithShort("Show diff across workspace repositories"),
		cmds.WithLong(`Show unified diff of changes across all repositories in the workspace.
This provides a consolidated view of all modifications in your multi-repository development.`),
		cmds.WithFlags(
			fields.New(
				"staged",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Show staged changes only"),
			),
			fields.New(
				"repo",
				fields.TypeString,
				fields.WithHelp("Show diff for specific repository only"),
			),
			fields.New(
				"jobs",
				fields.TypeInteger,
				fields.WithDefault(1),
				fields.WithHelp("Maximum concurrent repositories to process"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &DiffCommand{CommandDescription: desc}, nil
}

func (c *DiffCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	output.PrintHeader("Showing diff for workspace: %s", result.Workspace.Name)
	if result.Staged {
		output.PrintInfo("  (staged changes only)")
	}
	if result.RepoFilter != "" {
		output.PrintInfo("  (repository: %s)", result.RepoFilter)
	}
	fmt.Println()

	if !result.HasChanges {
		output.PrintInfo("No changes found in workspace.")
	} else {
		fmt.Println(result.Diff)
	}

	return nil
}

func (c *DiffCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	row := diffResultToRow(result)
	return gp.AddRow(ctx, row)
}

func (c *DiffCommand) execute(ctx context.Context, vals *values.Values) (*diffExecutionResult, error) {
	settings_ := &DiffSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode diff settings")
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect current workspace")
	}

	gitOps := wsm.NewGitOperations(workspace)
	diff, err := gitOps.GetDiffWithOptions(ctx, settings_.Staged, settings_.Repo, wsm.DiffOptions{MaxJobs: settings_.Jobs})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get diff")
	}

	noChanges := diff == "" || diff == "No changes found in workspace."

	return &diffExecutionResult{
		Workspace:  workspace,
		Diff:       diff,
		Staged:     settings_.Staged,
		RepoFilter: settings_.Repo,
		Jobs:       settings_.Jobs,
		HasChanges: !noChanges,
	}, nil
}

func diffResultToRow(result *diffExecutionResult) types.Row {
	return types.NewRow(
		types.MRP("workspace", result.Workspace.Name),
		types.MRP("workspace_path", result.Workspace.Path),
		types.MRP("staged", result.Staged),
		types.MRP("repo_filter", result.RepoFilter),
		types.MRP("jobs", result.Jobs),
		types.MRP("has_changes", result.HasChanges),
		types.MRP("diff", result.Diff),
	)
}

func NewDiffCobraCommand() (*cobra.Command, error) {
	command, err := NewDiffCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build diff command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
