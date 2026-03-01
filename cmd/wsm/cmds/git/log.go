package git

import (
	"context"
	"fmt"
	"sort"

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
var _ cmds.GlazeCommand = &LogCommand{}

type logExecutionResult struct {
	Workspace *wsm.Workspace
	Since     string
	Oneline   bool
	Limit     int
	Logs      map[string]string
}

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
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	output.PrintHeader("Commit history for workspace: %s", result.Workspace.Name)
	if result.Since != "" {
		output.PrintInfo("  (since: %s)", result.Since)
	}
	fmt.Println()

	if len(result.Logs) == 0 {
		output.PrintInfo("No commits found in workspace.")
		return nil
	}

	repoNames := make([]string, 0, len(result.Logs))
	for repoName := range result.Logs {
		repoNames = append(repoNames, repoName)
	}
	sort.Strings(repoNames)

	for _, repoName := range repoNames {
		repoLog := result.Logs[repoName]
		if repoLog == "" {
			continue
		}
		output.PrintHeader("=== Repository: %s ===", repoName)
		fmt.Println(repoLog)
		fmt.Println()
	}

	return nil
}

func (c *LogCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	for _, row := range logResultToRows(result) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add log row")
		}
	}

	return nil
}

func (c *LogCommand) execute(ctx context.Context, vals *values.Values) (*logExecutionResult, error) {
	settings_ := &LogSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode log settings")
	}

	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect current workspace")
	}

	historyOps := wsm.NewHistoryOperations(workspace)
	logs, err := historyOps.GetWorkspaceLog(ctx, settings_.Since, settings_.Oneline, settings_.Limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace log")
	}

	return &logExecutionResult{
		Workspace: workspace,
		Since:     settings_.Since,
		Oneline:   settings_.Oneline,
		Limit:     settings_.Limit,
		Logs:      logs,
	}, nil
}

func logResultToRows(result *logExecutionResult) []types.Row {
	if len(result.Logs) == 0 {
		return []types.Row{types.NewRow(
			types.MRP("workspace", result.Workspace.Name),
			types.MRP("workspace_path", result.Workspace.Path),
			types.MRP("since", result.Since),
			types.MRP("oneline", result.Oneline),
			types.MRP("limit", result.Limit),
			types.MRP("repository", ""),
			types.MRP("has_log", false),
			types.MRP("log", ""),
		)}
	}

	repoNames := make([]string, 0, len(result.Logs))
	for repoName := range result.Logs {
		repoNames = append(repoNames, repoName)
	}
	sort.Strings(repoNames)

	rows := make([]types.Row, 0, len(repoNames))
	for _, repoName := range repoNames {
		repoLog := result.Logs[repoName]
		rows = append(rows, types.NewRow(
			types.MRP("workspace", result.Workspace.Name),
			types.MRP("workspace_path", result.Workspace.Path),
			types.MRP("since", result.Since),
			types.MRP("oneline", result.Oneline),
			types.MRP("limit", result.Limit),
			types.MRP("repository", repoName),
			types.MRP("has_log", repoLog != ""),
			types.MRP("log", repoLog),
		))
	}

	return rows
}

func NewLogCobraCommand() (*cobra.Command, error) {
	command, err := NewLogCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build log command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
