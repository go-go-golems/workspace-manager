package js

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
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
	settings_ := &RunnerSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode runner settings")
	}
	if settings_.ScriptPath == "" {
		return errors.New("script path is required")
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	result, err := wsmjsrunner.RunFile(ctx, settings_.ScriptPath)
	if err != nil {
		return errors.Wrap(err, "failed to execute JavaScript")
	}

	if wsmcmdcommon.ShouldOutputHuman(mode) {
		output.PrintSuccess("Executed JS script: %s", settings_.ScriptPath)
		if settings_.PrintResult && result != nil {
			encoded, encErr := json.MarshalIndent(result, "", "  ")
			if encErr != nil {
				fmt.Printf("Result: %v\n", result)
			} else {
				fmt.Println(string(encoded))
			}
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		rows := []types.Row{types.NewRow(
			types.MRP("script", settings_.ScriptPath),
			types.MRP("result", result),
			types.MRP("has_result", result != nil),
			types.MRP("status", "ok"),
		)}
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit runner rows")
		}
	}

	return nil
}

func NewRunnerCobraCommand() (*cobra.Command, error) {
	command, err := NewRunnerCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build runner command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommand(command)
}
