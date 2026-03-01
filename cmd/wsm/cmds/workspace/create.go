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

// CreateCommand creates a new multi-repository workspace.
type CreateCommand struct {
	*cmds.CommandDescription
}

// CreateSettings stores parsed create command settings.
type CreateSettings struct {
	WorkspaceNameArg string   `glazed:"workspace-name"`
	Repos            []string `glazed:"repos"`
	Branch           string   `glazed:"branch"`
	BranchPrefix     string   `glazed:"branch-prefix"`
	BaseBranch       string   `glazed:"base-branch"`
	AgentSource      string   `glazed:"agent-source"`
	Interactive      bool     `glazed:"interactive"`
	DryRun           bool     `glazed:"dry-run"`
}

var _ cmds.BareCommand = &CreateCommand{}
var _ cmds.GlazeCommand = &CreateCommand{}

type createExecutionResult struct {
	Workspace           *wsm.Workspace
	FinalBranch         string
	AutoBranchGenerated bool
	DryRun              bool
	Interactive         bool
	Cancelled           bool
}

func NewCreateCommand() (*CreateCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"create",
		cmds.WithShort("Create a new multi-repository workspace"),
		cmds.WithLong(`Create a new workspace with specified repositories.

If no branch is specified, a branch is auto-generated as:
  <branch-prefix>/<workspace-name>`),
		cmds.WithFlags(
			fields.New("workspace-name", fields.TypeString, fields.WithIsArgument(true), fields.WithHelp("Workspace name")),
			fields.New("repos", fields.TypeStringList, fields.WithHelp("Repository names to include (comma-separated)")),
			fields.New("branch", fields.TypeString, fields.WithHelp("Branch name for worktrees")),
			fields.New("branch-prefix", fields.TypeString, fields.WithDefault("task"), fields.WithHelp("Prefix for auto-generated branch names")),
			fields.New("base-branch", fields.TypeString, fields.WithHelp("Base branch to create new branch from")),
			fields.New("agent-source", fields.TypeString, fields.WithHelp("Path to AGENT.md template file")),
			fields.New("interactive", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Interactive repository selection")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Show what would be created without creating")),
		),
	)
	if err != nil {
		return nil, err
	}
	return &CreateCommand{CommandDescription: desc}, nil
}

func (c *CreateCommand) Run(ctx context.Context, vals *values.Values) error {
	result, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	if result.Cancelled {
		output.PrintInfo("Operation cancelled.")
		return nil
	}

	workspace := result.Workspace

	if result.AutoBranchGenerated {
		output.PrintInfo("Using auto-generated branch: %s", result.FinalBranch)
	}
	if result.DryRun {
		if err := showWorkspacePreview(workspace); err != nil {
			return err
		}
	} else {
		output.PrintSuccess("Workspace '%s' created successfully!", workspace.Name)
		fmt.Println()
		output.PrintHeader("Workspace Details")
		fmt.Printf("  Path: %s\n", workspace.Path)
		fmt.Printf("  Repositories: %s\n", strings.Join(getRepositoryNames(workspace.Repositories), ", "))
		if workspace.Branch != "" {
			fmt.Printf("  Branch: %s\n", workspace.Branch)
		}
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

func (c *CreateCommand) RunIntoGlazeProcessor(
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

	row := createResultToRow(result)
	return gp.AddRow(ctx, row)
}

func (c *CreateCommand) execute(ctx context.Context, vals *values.Values) (*createExecutionResult, error) {
	settings_ := &CreateSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, errors.Wrap(err, "failed to decode create settings")
	}

	if settings_.WorkspaceNameArg == "" {
		return nil, errors.New("workspace name is required")
	}

	repos := settings_.Repos
	if settings_.Interactive {
		wm, err := wsm.NewWorkspaceManager()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create workspace manager")
		}
		selectedRepos, err := selectRepositoriesInteractively(wm)
		if err != nil {
			if isUserCancelledError(err) {
				return &createExecutionResult{Cancelled: true}, nil
			}
			return nil, errors.Wrap(err, "interactive selection failed")
		}
		repos = selectedRepos
	}

	workflow, err := workflows.NewCreateWorkflow()
	if err != nil {
		return nil, err
	}

	workflowResult, err := workflow.Create(ctx, workflows.CreateRequest{
		Name:         settings_.WorkspaceNameArg,
		Repos:        repos,
		Branch:       settings_.Branch,
		BranchPrefix: settings_.BranchPrefix,
		BaseBranch:   settings_.BaseBranch,
		AgentSource:  settings_.AgentSource,
		DryRun:       settings_.DryRun,
	})
	if err != nil {
		if isUserCancelledError(err) {
			return &createExecutionResult{Cancelled: true}, nil
		}
		return nil, err
	}

	return &createExecutionResult{
		Workspace:           workflowResult.Workspace,
		FinalBranch:         workflowResult.FinalBranch,
		AutoBranchGenerated: workflowResult.AutoBranchGenerated,
		DryRun:              settings_.DryRun,
		Interactive:         settings_.Interactive,
	}, nil
}

func createResultToRow(result *createExecutionResult) types.Row {
	repoNames := getRepositoryNames(result.Workspace.Repositories)
	return types.NewRow(
		types.MRP("workspace", result.Workspace.Name),
		types.MRP("workspace_path", result.Workspace.Path),
		types.MRP("repositories", repoNames),
		types.MRP("repository_count", len(repoNames)),
		types.MRP("branch", result.Workspace.Branch),
		types.MRP("base_branch", result.Workspace.BaseBranch),
		types.MRP("go_workspace", result.Workspace.GoWorkspace),
		types.MRP("agent_md", result.Workspace.AgentMD),
		types.MRP("dry_run", result.DryRun),
		types.MRP("auto_branch_generated", result.AutoBranchGenerated),
		types.MRP("final_branch", result.FinalBranch),
		types.MRP("interactive", result.Interactive),
	)
}

func NewCreateCobraCommand() (*cobra.Command, error) {
	command, err := NewCreateCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build create command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
