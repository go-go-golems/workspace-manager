package git

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
	settings_ := &DiffSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode diff settings")
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	gitOps := wsm.NewGitOperations(workspace)
	diff, err := gitOps.GetDiffWithOptions(ctx, settings_.Staged, settings_.Repo, wsm.DiffOptions{MaxJobs: settings_.Jobs})
	if err != nil {
		return errors.Wrap(err, "failed to get diff")
	}

	noChanges := diff == "" || diff == "No changes found in workspace."

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		output.PrintHeader("Showing diff for workspace: %s", workspace.Name)
		if settings_.Staged {
			output.PrintInfo("  (staged changes only)")
		}
		if settings_.Repo != "" {
			output.PrintInfo("  (repository: %s)", settings_.Repo)
		}
		fmt.Println()

		if noChanges {
			output.PrintInfo("No changes found in workspace.")
		} else {
			fmt.Println(diff)
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		rows := []types.Row{types.NewRow(
			types.MRP("workspace", workspace.Name),
			types.MRP("workspace_path", workspace.Path),
			types.MRP("staged", settings_.Staged),
			types.MRP("repo_filter", settings_.Repo),
			types.MRP("jobs", settings_.Jobs),
			types.MRP("has_changes", !noChanges),
			types.MRP("diff", diff),
		)}
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit diff rows")
		}
	}

	return nil
}

func NewDiffCobraCommand() (*cobra.Command, error) {
	command, err := NewDiffCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build diff command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommand(command)
}
