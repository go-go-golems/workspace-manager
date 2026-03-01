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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// AddCommand adds a repository to an existing workspace.
type AddCommand struct {
	*cmds.CommandDescription
}

// AddSettings stores parsed add command settings.
type AddSettings struct {
	WorkspaceNameArg string `glazed:"workspace-name"`
	RepoNameArg      string `glazed:"repo-name"`
	WorkspaceName    string `glazed:"workspace"`
	RepoName         string `glazed:"repo"`
	Branch           string `glazed:"branch"`
	Force            bool   `glazed:"force"`
}

var _ cmds.BareCommand = &AddCommand{}
var _ cmds.GlazeCommand = &AddCommand{}

type addExecutionResult struct {
	WorkspaceName string
	RepoName      string
	Branch        string
	Force         bool
}

func NewAddCommand() (*AddCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"add",
		cmds.WithShort("Add a repository to an existing workspace"),
		cmds.WithLong(`Add a repository to an existing workspace and create the necessary worktree.

Examples:
  wsm add my-workspace my-repo
  wsm add my-workspace my-repo --branch feature/override
  wsm add --workspace my-workspace --repo my-repo --force`),
		cmds.WithFlags(
			fields.New(
				"workspace-name",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Workspace name (positional)"),
			),
			fields.New(
				"repo-name",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Repository name (positional)"),
			),
			fields.New(
				"workspace",
				fields.TypeString,
				fields.WithHelp("Workspace name"),
			),
			fields.New(
				"repo",
				fields.TypeString,
				fields.WithHelp("Repository name"),
			),
			fields.New(
				"branch",
				fields.TypeString,
				fields.WithShortFlag("b"),
				fields.WithHelp("Branch name to use (defaults to workspace branch)"),
			),
			fields.New(
				"force",
				fields.TypeBool,
				fields.WithShortFlag("f"),
				fields.WithDefault(false),
				fields.WithHelp("Force overwrite if branch already exists"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &AddCommand{CommandDescription: desc}, nil
}

func (c *AddCommand) Run(ctx context.Context, vals *values.Values) error {
	_, err := c.execute(ctx, vals)
	return err
}

func (c *AddCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	row := addResultToRow(result)
	return gp.AddRow(ctx, row)
}

func (c *AddCommand) execute(ctx context.Context, vals *values.Values) (*addExecutionResult, error) {
	settings_ := &AddSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode add settings")
	}

	workspaceName := settings_.WorkspaceName
	if settings_.WorkspaceNameArg != "" {
		workspaceName = settings_.WorkspaceNameArg
	}
	if workspaceName == "" {
		return nil, errors.New("workspace name is required (positional <workspace-name> or --workspace)")
	}

	repoName := settings_.RepoName
	if settings_.RepoNameArg != "" {
		repoName = settings_.RepoNameArg
	}
	if repoName == "" {
		return nil, errors.New("repository name is required (positional <repo-name> or --repo)")
	}

	wm, err := wsm.NewWorkspaceManager()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace manager")
	}

	if err := wm.AddRepositoryToWorkspace(ctx, workspaceName, repoName, settings_.Branch, settings_.Force); err != nil {
		return nil, err
	}

	return &addExecutionResult{
		WorkspaceName: workspaceName,
		RepoName:      repoName,
		Branch:        settings_.Branch,
		Force:         settings_.Force,
	}, nil
}

func addResultToRow(result *addExecutionResult) types.Row {
	return types.NewRow(
		types.MRP("workspace", result.WorkspaceName),
		types.MRP("repository", result.RepoName),
		types.MRP("branch", result.Branch),
		types.MRP("force", result.Force),
		types.MRP("status", "added"),
	)
}

func NewAddCobraCommand() (*cobra.Command, error) {
	command, err := NewAddCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build add command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
