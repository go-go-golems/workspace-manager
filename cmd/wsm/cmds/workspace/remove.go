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

// RemoveCommand removes a repository from an existing workspace.
type RemoveCommand struct {
	*cmds.CommandDescription
}

// RemoveSettings stores parsed remove command settings.
type RemoveSettings struct {
	WorkspaceNameArg string `glazed:"workspace-name"`
	RepoNameArg      string `glazed:"repo-name"`
	WorkspaceName    string `glazed:"workspace"`
	RepoName         string `glazed:"repo"`
	Force            bool   `glazed:"force"`
	RemoveFiles      bool   `glazed:"remove-files"`
}

var _ cmds.BareCommand = &RemoveCommand{}
var _ cmds.GlazeCommand = &RemoveCommand{}

type removeExecutionResult struct {
	WorkspaceName string
	RepoName      string
	Force         bool
	RemoveFiles   bool
}

func NewRemoveCommand() (*RemoveCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"remove",
		cmds.WithShort("Remove a repository from an existing workspace"),
		cmds.WithLong(`Remove a repository from an existing workspace and clean up its worktree.

Examples:
  wsm remove my-workspace my-repo
  wsm remove my-workspace my-repo --force
  wsm remove --workspace my-workspace --repo my-repo --remove-files`),
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
				"force",
				fields.TypeBool,
				fields.WithShortFlag("f"),
				fields.WithDefault(false),
				fields.WithHelp("Force remove worktree even with uncommitted changes"),
			),
			fields.New(
				"remove-files",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Remove the repository directory from workspace"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &RemoveCommand{CommandDescription: desc}, nil
}

func (c *RemoveCommand) Run(ctx context.Context, vals *values.Values) error {
	_, err := c.execute(ctx, vals)
	return err
}

func (c *RemoveCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	row := removeResultToRow(result)
	return gp.AddRow(ctx, row)
}

func (c *RemoveCommand) execute(ctx context.Context, vals *values.Values) (*removeExecutionResult, error) {
	settings_ := &RemoveSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode remove settings")
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

	if err := wm.RemoveRepositoryFromWorkspace(ctx, workspaceName, repoName, settings_.Force, settings_.RemoveFiles); err != nil {
		return nil, err
	}

	return &removeExecutionResult{
		WorkspaceName: workspaceName,
		RepoName:      repoName,
		Force:         settings_.Force,
		RemoveFiles:   settings_.RemoveFiles,
	}, nil
}

func removeResultToRow(result *removeExecutionResult) types.Row {
	return types.NewRow(
		types.MRP("workspace", result.WorkspaceName),
		types.MRP("repository", result.RepoName),
		types.MRP("force", result.Force),
		types.MRP("remove_files", result.RemoveFiles),
		types.MRP("status", "removed"),
	)
}

func NewRemoveCobraCommand() (*cobra.Command, error) {
	command, err := NewRemoveCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build remove command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
