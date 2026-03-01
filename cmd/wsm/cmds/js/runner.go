package js

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	wsmcmdcommon "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/common"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	wsmjsrunner "github.com/go-go-golems/workspace-manager/pkg/wsmjs/runner"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RunnerCommand runs JavaScript files with the wsm module pre-registered.
type RunnerCommand struct {
	*cmds.CommandDescription
}

// RunnerSettings stores parsed runner command settings.
type RunnerSettings struct {
	ScriptPath  string `glazed:"script"`
	PrintResult bool   `glazed:"print-result"`
}

var _ cmds.BareCommand = &RunnerCommand{}
var _ cmds.GlazeCommand = &RunnerCommand{}

type runnerExecutionResult struct {
	ScriptPath  string
	PrintResult bool
	Result      interface{}
}

func NewRunnerCommand() (*RunnerCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"runner",
		cmds.WithShort("Run JavaScript with the wsm API preloaded"),
		cmds.WithLong(`Execute a JavaScript file with require("wsm") available.

Example:
  wsm runner ./demo/js/wsm-smoke.js`),
		cmds.WithFlags(
			fields.New(
				"script",
				fields.TypeString,
				fields.WithIsArgument(true),
				fields.WithHelp("Path to JavaScript file to execute"),
			),
			fields.New(
				"print-result",
				fields.TypeBool,
				fields.WithDefault(true),
				fields.WithHelp("Print script return value in human mode"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &RunnerCommand{CommandDescription: desc}, nil
}

func (c *RunnerCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	output.PrintSuccess("Executed JS script: %s", result.ScriptPath)
	if result.PrintResult && result.Result != nil {
		encoded, encErr := json.MarshalIndent(result.Result, "", "  ")
		if encErr != nil {
			fmt.Printf("Result: %v\n", result.Result)
		} else {
			fmt.Println(string(encoded))
		}
	}

	return nil
}

func (c *RunnerCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	row := runnerResultToRow(result)
	return gp.AddRow(ctx, row)
}

func (c *RunnerCommand) execute(ctx context.Context, vals *values.Values) (*runnerExecutionResult, error) {
	settings_ := &RunnerSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode runner settings")
	}
	if settings_.ScriptPath == "" {
		return nil, errors.New("script path is required")
	}

	scriptResult, err := wsmjsrunner.RunFile(ctx, settings_.ScriptPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute JavaScript")
	}

	return &runnerExecutionResult{
		ScriptPath:  settings_.ScriptPath,
		PrintResult: settings_.PrintResult,
		Result:      scriptResult,
	}, nil
}

func runnerResultToRow(result *runnerExecutionResult) types.Row {
	return types.NewRow(
		types.MRP("script", result.ScriptPath),
		types.MRP("result", result.Result),
		types.MRP("has_result", result.Result != nil),
		types.MRP("status", "ok"),
	)
}

func NewRunnerCobraCommand() (*cobra.Command, error) {
	command, err := NewRunnerCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build runner command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
