package workspace

import (
	"context"
	"fmt"
	"os"

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

// SetBaseCommand sets a per-repo comparison-base override.
type SetBaseCommand struct {
	*cmds.CommandDescription
}

// SetBaseSettings stores parsed set-base command settings.
type SetBaseSettings struct {
	RepoNameArg    string `glazed:"repo-name"`
	WorkspaceName  string `glazed:"workspace"`
	WorkspaceNameX string `glazed:"workspace-name"` // positional alias
	Branch         string `glazed:"branch"`
	Remote         string `glazed:"remote"`
	Global         bool   `glazed:"global"`
	Fetch          bool   `glazed:"fetch"`
}

var _ cmds.BareCommand = &SetBaseCommand{}
var _ cmds.GlazeCommand = &SetBaseCommand{}

type setBaseExecutionResult struct {
	WorkspaceName string
	RepoName      string
	Branch        string
	Remote        string
	Global        bool
	Fetched       bool
}

func NewSetBaseCommand() (*SetBaseCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"set-base",
		cmds.WithShort("Set the comparison base branch for one repo in a workspace"),
		cmds.WithLong(`Set the base (upstream) branch used by 'wsm status' to compute
is_merged/needs_rebase for a single repository.

By default the override is written to the in-workspace .wsm/wsm.json
(workspace-specific). Use --global to write the config-dir workspace JSON
instead. The in-workspace override takes precedence at load time (local beats
global); the two stores are never mirrored.

Use --fetch to 'git fetch <remote> <branch>' so the remote-tracking ref
exists before the next status check.`),
		cmds.WithFlags(
			fields.New("repo-name", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Repository name (positional)")),
			fields.New("workspace", fields.TypeString, fields.WithHelp("Workspace name (detected from cwd if empty)")),
			fields.New("workspace-name", fields.TypeString, fields.WithHelp("Workspace name (alias for --workspace)")),
			fields.New("branch", fields.TypeString, fields.WithHelp("Base branch to compare against (required)")),
			fields.New("remote", fields.TypeString, fields.WithHelp("Remote for the base (default origin)")),
			fields.New("global", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Write the config-dir workspace JSON instead of the in-workspace .wsm/wsm.json")),
			fields.New("fetch", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("git fetch <remote> <branch> in the worktree first")),
		),
	)
	if err != nil {
		return nil, err
	}
	return &SetBaseCommand{CommandDescription: desc}, nil
}

func (c *SetBaseCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}
	store := "workspace"
	if result.Global {
		store = "global"
	}
	remote := result.Remote
	if remote == "" {
		remote = "origin"
	}
	output.PrintSuccess("Set %s base to %s (remote: %s) [stored: %s]",
		result.RepoName, result.Branch, remote, store)
	return nil
}

func (c *SetBaseCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}
	row := setBaseResultToRow(result)
	return gp.AddRow(ctx, row)
}

func (c *SetBaseCommand) execute(ctx context.Context, vals *values.Values) (*setBaseExecutionResult, error) {
	settings_ := &SetBaseSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode set-base settings")
	}

	repoName := settings_.RepoNameArg
	if repoName == "" {
		return nil, errors.New("repository name is required (positional <repo-name>)")
	}

	workspaceName := settings_.WorkspaceName
	if settings_.WorkspaceNameX != "" {
		workspaceName = settings_.WorkspaceNameX
	}
	if workspaceName == "" {
		// Detect from cwd like other commands.
		wcs := wsm.NewWorkspaceContextService()
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current directory")
		}
		detected, err := wcs.DetectWorkspaceName(cwd)
		if err != nil {
			return nil, errors.Wrap(err, "failed to detect workspace. Use 'wsm set-base <repo> --workspace <ws> --branch <branch>' or specify --workspace")
		}
		workspaceName = detected
	}

	if settings_.Branch == "" {
		return nil, errors.New("--branch is required")
	}

	wm, err := wsm.NewWorkspaceManager()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace manager")
	}

	if err := wm.SetRepoBase(ctx, workspaceName, wsm.SetRepoBaseOptions{
		RepoName: repoName,
		Branch:   settings_.Branch,
		Remote:   settings_.Remote,
		Global:   settings_.Global,
		Fetch:    settings_.Fetch,
	}); err != nil {
		return nil, err
	}

	return &setBaseExecutionResult{
		WorkspaceName: workspaceName,
		RepoName:      repoName,
		Branch:        settings_.Branch,
		Remote:        settings_.Remote,
		Global:        settings_.Global,
		Fetched:       settings_.Fetch,
	}, nil
}

func setBaseResultToRow(result *setBaseExecutionResult) types.Row {
	store := "workspace"
	if result.Global {
		store = "global"
	}
	remote := result.Remote
	if remote == "" {
		remote = "origin"
	}
	return types.NewRow(
		types.MRP("workspace", result.WorkspaceName),
		types.MRP("repository", result.RepoName),
		types.MRP("base_branch", result.Branch),
		types.MRP("base_remote", remote),
		types.MRP("stored", store),
		types.MRP("fetched", result.Fetched),
		types.MRP("status", "set"),
	)
}

func NewSetBaseCobraCommand() (*cobra.Command, error) {
	command, err := NewSetBaseCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build set-base command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
