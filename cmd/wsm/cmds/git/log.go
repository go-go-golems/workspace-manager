package git

import (
	"context"
	"fmt"
	"sort"

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

// LogCommand shows commit history across workspace repositories.
type LogCommand struct {
	*cmds.CommandDescription
}

// LogSettings stores parsed log command settings.
type LogSettings struct {
	Since   string `glazed:"since"`
	Oneline bool   `glazed:"oneline"`
	Limit   int    `glazed:"limit"`
}

var _ cmds.BareCommand = &LogCommand{}

func NewLogCommand() (*LogCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"log",
		cmds.WithShort("Show commit history across workspace repositories"),
		cmds.WithLong(`Show commit history spanning multiple repositories in the workspace.
This provides a unified view of development activity across your projects.`),
		cmds.WithFlags(
			fields.New(
				"since",
				fields.TypeString,
				fields.WithHelp("Show commits since date (for example, '1 week ago')"),
			),
			fields.New(
				"oneline",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Show one line per commit"),
			),
			fields.New(
				"limit",
				fields.TypeInteger,
				fields.WithDefault(10),
				fields.WithHelp("Limit number of commits per repository"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &LogCommand{CommandDescription: desc}, nil
}

func (c *LogCommand) Run(ctx context.Context, vals *values.Values) error {
	settings_ := &LogSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode log settings")
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	historyOps := wsm.NewHistoryOperations(workspace)
	logs, err := historyOps.GetWorkspaceLog(ctx, settings_.Since, settings_.Oneline, settings_.Limit)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace log")
	}

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		output.PrintHeader("Commit history for workspace: %s", workspace.Name)
		if settings_.Since != "" {
			output.PrintInfo("  (since: %s)", settings_.Since)
		}
		fmt.Println()

		if len(logs) == 0 {
			output.PrintInfo("No commits found in workspace.")
		} else {
			repoNames := make([]string, 0, len(logs))
			for repoName := range logs {
				repoNames = append(repoNames, repoName)
			}
			sort.Strings(repoNames)

			for _, repoName := range repoNames {
				repoLog := logs[repoName]
				if repoLog == "" {
					continue
				}
				output.PrintHeader("=== Repository: %s ===", repoName)
				fmt.Println(repoLog)
				fmt.Println()
			}
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		if len(logs) == 0 {
			rows := []types.Row{types.NewRow(
				types.MRP("workspace", workspace.Name),
				types.MRP("workspace_path", workspace.Path),
				types.MRP("since", settings_.Since),
				types.MRP("oneline", settings_.Oneline),
				types.MRP("limit", settings_.Limit),
				types.MRP("repository", ""),
				types.MRP("has_log", false),
				types.MRP("log", ""),
			)}
			if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
				return errors.Wrap(err, "failed to emit empty log rows")
			}
			return nil
		}

		repoNames := make([]string, 0, len(logs))
		for repoName := range logs {
			repoNames = append(repoNames, repoName)
		}
		sort.Strings(repoNames)

		rows := make([]types.Row, 0, len(repoNames))
		for _, repoName := range repoNames {
			repoLog := logs[repoName]
			rows = append(rows, types.NewRow(
				types.MRP("workspace", workspace.Name),
				types.MRP("workspace_path", workspace.Path),
				types.MRP("since", settings_.Since),
				types.MRP("oneline", settings_.Oneline),
				types.MRP("limit", settings_.Limit),
				types.MRP("repository", repoName),
				types.MRP("has_log", repoLog != ""),
				types.MRP("log", repoLog),
			))
		}

		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit log rows")
		}
	}

	return nil
}

func NewLogCobraCommand() (*cobra.Command, error) {
	command, err := NewLogCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build log command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommand(command)
}
