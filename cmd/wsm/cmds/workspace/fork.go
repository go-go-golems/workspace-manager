package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	wsmcmdcommon "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/common"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ForkCommand creates a new workspace by forking an existing workspace.
type ForkCommand struct {
	*cmds.CommandDescription
}

// ForkSettings stores parsed fork command settings.
type ForkSettings struct {
	NewWorkspaceNameArg    string `glazed:"new-workspace-name"`
	SourceWorkspaceNameArg string `glazed:"source-workspace-name"`
	SourceWorkspaceName    string `glazed:"workspace"`
	Branch                 string `glazed:"branch"`
	BranchPrefix           string `glazed:"branch-prefix"`
	AgentSource            string `glazed:"agent-source"`
	DryRun                 bool   `glazed:"dry-run"`
}

var _ cmds.BareCommand = &ForkCommand{}
var _ cmds.GlazeCommand = &ForkCommand{}

type forkExecutionResult struct {
	Workspace           *wsm.Workspace
	Plan                *workflows.ForkPlan
	BranchProvided      bool
	AgentSourceProvided bool
	DryRun              bool
	Cancelled           bool
}

func NewForkCommand() (*ForkCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"fork",
		cmds.WithShort("Create a new workspace by forking an existing workspace"),
		cmds.WithLong(`Create a new workspace that is a fork of an existing workspace.
If source workspace is not provided, detects from current workspace.`),
		cmds.WithFlags(
			fields.New("new-workspace-name", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("New workspace name")),
			fields.New("source-workspace-name", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Source workspace name")),
			fields.New("workspace", fields.TypeString, fields.WithHelp("Source workspace name")),
			fields.New("branch", fields.TypeString, fields.WithHelp("Branch name for the new workspace")),
			fields.New("branch-prefix", fields.TypeString, fields.WithDefault("task"), fields.WithHelp("Prefix for auto-generated branch names")),
			fields.New("agent-source", fields.TypeString, fields.WithHelp("Path to AGENT.md template file")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Show what would be created without creating")),
		),
	)
	if err != nil {
		return nil, err
	}
	return &ForkCommand{CommandDescription: desc}, nil
}

func (c *ForkCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	if result.Cancelled {
		output.PrintInfo("Operation cancelled.")
		return nil
	}

	plan := result.Plan
	workspace := result.Workspace

	output.PrintInfo("Forking workspace '%s' to create '%s'", plan.SourceWorkspace.Name, workspace.Name)
	output.PrintInfo("Using base branch: %s", plan.BaseBranch)
	if !result.BranchProvided {
		output.PrintInfo("Using auto-generated branch: %s", plan.FinalBranch)
	}
	if !result.AgentSourceProvided && plan.FinalAgentSource != "" {
		output.PrintInfo("Using AGENT.md from source workspace: %s", plan.FinalAgentSource)
	}

	if result.DryRun {
		output.PrintHeader("Fork Preview: %s -> %s", plan.SourceWorkspace.Name, workspace.Name)
		fmt.Println()
		output.PrintInfo("Source workspace:")
		fmt.Printf("  Name: %s\n", plan.SourceWorkspace.Name)
		fmt.Printf("  Path: %s\n", plan.SourceWorkspace.Path)
		fmt.Printf("  Current branch: %s\n", plan.BaseBranch)
		fmt.Println()
		if err := showWorkspacePreview(workspace); err != nil {
			return err
		}
	} else {
		output.PrintSuccess("Workspace '%s' forked successfully from '%s'!", workspace.Name, plan.SourceWorkspace.Name)
		fmt.Println()
		output.PrintHeader("Fork Details")
		fmt.Printf("  Source: %s (branch: %s)\n", plan.SourceWorkspace.Name, plan.BaseBranch)
		fmt.Printf("  New workspace: %s\n", workspace.Name)
		fmt.Printf("  Path: %s\n", workspace.Path)
		fmt.Printf("  Repositories: %s\n", strings.Join(getRepositoryNames(workspace.Repositories), ", "))
		fmt.Printf("  New branch: %s\n", workspace.Branch)
		fmt.Printf("  Base branch: %s\n", plan.BaseBranch)
		if workspace.GoWorkspace {
			fmt.Printf("  Go workspace: yes (go.work created)\n")
		}
		if workspace.AgentMD != "" {
			fmt.Printf("  AGENT.md: copied from %s\n", workspace.AgentMD)
		}
		fmt.Println()
		output.PrintInfo("To start working:")
		fmt.Printf("  cd %s\n", workspace.Path)
	}

	return nil
}

func (c *ForkCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	if result.Cancelled {
		return nil
	}

	row := forkResultToRow(result)
	return gp.AddRow(ctx, row)
}

func (c *ForkCommand) execute(ctx context.Context, vals *values.Values) (*forkExecutionResult, error) {
	settings_ := &ForkSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode fork settings")
	}

	if settings_.NewWorkspaceNameArg == "" {
		return nil, errors.New("new workspace name is required")
	}

	sourceWorkspaceName := settings_.SourceWorkspaceName
	if settings_.SourceWorkspaceNameArg != "" {
		sourceWorkspaceName = settings_.SourceWorkspaceNameArg
	}

	workflow, err := workflows.NewForkWorkflow()
	if err != nil {
		return nil, err
	}

	req := workflows.ForkRequest{
		NewWorkspaceName:    settings_.NewWorkspaceNameArg,
		SourceWorkspaceName: sourceWorkspaceName,
		Branch:              settings_.Branch,
		BranchPrefix:        settings_.BranchPrefix,
		AgentSource:         settings_.AgentSource,
		DryRun:              settings_.DryRun,
	}

	plan, err := workflow.Plan(ctx, req)
	if err != nil {
		return nil, err
	}

	workspace, _, err := workflow.Fork(ctx, req)
	if err != nil {
		if isUserCancelledError(err) {
			return &forkExecutionResult{Cancelled: true}, nil
		}
		return nil, err
	}

	return &forkExecutionResult{
		Workspace:           workspace,
		Plan:                plan,
		BranchProvided:      settings_.Branch != "",
		AgentSourceProvided: settings_.AgentSource != "",
		DryRun:              settings_.DryRun,
	}, nil
}

func forkResultToRow(result *forkExecutionResult) types.Row {
	repoNames := getRepositoryNames(result.Workspace.Repositories)
	return types.NewRow(
		types.MRP("source_workspace", result.Plan.SourceWorkspace.Name),
		types.MRP("new_workspace", result.Workspace.Name),
		types.MRP("workspace_path", result.Workspace.Path),
		types.MRP("base_branch", result.Plan.BaseBranch),
		types.MRP("final_branch", result.Plan.FinalBranch),
		types.MRP("repositories", repoNames),
		types.MRP("repository_count", len(repoNames)),
		types.MRP("agent_source", result.Plan.FinalAgentSource),
		types.MRP("dry_run", result.DryRun),
	)
}

func NewForkCobraCommand() (*cobra.Command, error) {
	command, err := NewForkCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build fork command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
