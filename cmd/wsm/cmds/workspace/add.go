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
	settings_ := &AddSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode add settings")
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	workspaceName := settings_.WorkspaceName
	if settings_.WorkspaceNameArg != "" {
		workspaceName = settings_.WorkspaceNameArg
	}
	if workspaceName == "" {
		return errors.New("workspace name is required (positional <workspace-name> or --workspace)")
	}

	repoName := settings_.RepoName
	if settings_.RepoNameArg != "" {
		repoName = settings_.RepoNameArg
	}
	if repoName == "" {
		return errors.New("repository name is required (positional <repo-name> or --repo)")
	}

	wm, err := wsm.NewWorkspaceManager()
	if err != nil {
		return errors.Wrap(err, "failed to create workspace manager")
	}

	if err := wm.AddRepositoryToWorkspace(ctx, workspaceName, repoName, settings_.Branch, settings_.Force); err != nil {
		return err
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		rows := []types.Row{types.NewRow(
			types.MRP("workspace", workspaceName),
			types.MRP("repository", repoName),
			types.MRP("branch", settings_.Branch),
			types.MRP("force", settings_.Force),
			types.MRP("status", "added"),
		)}
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit add rows")
		}
	}

	return nil
}

func NewAddCobraCommand() (*cobra.Command, error) {
	command, err := NewAddCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build add command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommand(command)
}
