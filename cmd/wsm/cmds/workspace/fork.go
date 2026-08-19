package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	wsmcmdcommon "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/common"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	branch "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
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
	BaseBranch             string `glazed:"base-branch"`
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
			fields.New("base-branch", fields.TypeString, fields.WithHelp("Base/upstream branch to fork from (use when source repos are on different branches)")),
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
	result, err := c.execute(ctx, vals, true, true)
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
	result, err := c.execute(ctx, vals, false, false)
	if err != nil {
		return err
	}

	if result.Cancelled {
		return nil
	}

	row := forkResultToRow(result)
	return gp.AddRow(ctx, row)
}

func (c *ForkCommand) execute(ctx context.Context, vals *values.Values, emitHuman bool, allowPrompt bool) (*forkExecutionResult, error) {
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
		BaseBranch:          settings_.BaseBranch,
		AgentSource:         settings_.AgentSource,
		DryRun:              settings_.DryRun,
	}

	plan, err := workflow.Plan(ctx, req)
	if err != nil {
		// F1/F2: branch divergence -> prompt interactively, or require
		// --base-branch in non-interactive mode (mirrors delete's --force gate).
		var div *workflows.ErrBranchDivergence
		if errors.As(err, &div) {
			if !allowPrompt {
				return nil, errors.Errorf(
					"source workspace '%s' repos are on different branches (%s); pass --base-branch to choose one",
					div.Source, strings.Join(div.DistinctBranches(), ", "))
			}
			chosen, ok, cancelled := promptBaseBranch(div)
			if cancelled {
				return &forkExecutionResult{Cancelled: true}, nil
			}
			if !ok {
				return nil, errors.New("no base branch selected")
			}
			req.BaseBranch = chosen
			plan, err = workflow.Plan(ctx, req)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
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

// promptBaseBranch asks the user to choose a base branch when a fork's source
// repos are on different branches. It returns (chosen, ok, cancelled):
// cancelled=true if the user aborted; ok=false if the user did not confirm.
// The default is the most frequent observed branch; the conventional
// task/<source-name> is offered if not already among the observed branches.
func promptBaseBranch(div *workflows.ErrBranchDivergence) (string, bool, bool) {
	options := div.DistinctBranches()
	// Ensure the conventional expected branch is selectable even if no repo
	// is currently on it.
	if div.Expected != "" {
		present := false
		for _, o := range options {
			if o == div.Expected {
				present = true
				break
			}
		}
		if !present {
			options = append([]string{div.Expected}, options...)
		}
	}

	defaultBranch := branch.MostFrequentBranch(div.Branches)
	if defaultBranch == "" {
		defaultBranch = div.Expected
	}

	selected := defaultBranch
	// Run the selection first, then the confirmation as a separate form, so the
	// confirm title reflects the branch the user actually selected rather than
	// the default formatted before form.Run (huh evaluates the title once at
	// build time, so a single combined form would name the original default).
	selectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Source repos are on different branches. Choose the base branch to fork from:").
				Options(huh.NewOptions(options...)...).
				Value(&selected),
		),
	)
	if err := selectForm.Run(); err != nil {
		if isUserCancelledError(err) {
			return "", false, true
		}
		return "", false, false
	}

	var confirm bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Fork using base branch '%s'?", selected)).
				Description(showDivergence(div)).
				Value(&confirm),
		),
	)
	if err := confirmForm.Run(); err != nil {
		if isUserCancelledError(err) {
			return "", false, true
		}
		return "", false, false
	}
	return selected, confirm, false
}

// showDivergence renders the per-repo branch map so the user can see exactly
// which repo is on which branch before confirming.
func showDivergence(div *workflows.ErrBranchDivergence) string {
	repos := make([]string, 0, len(div.Branches))
	for name := range div.Branches {
		repos = append(repos, name)
	}
	sortStrings(repos)
	lines := make([]string, 0, len(repos))
	for _, name := range repos {
		lines = append(lines, fmt.Sprintf("  %s -> %s", name, div.Branches[name]))
	}
	return "Per-repo branches:\n" + strings.Join(lines, "\n")
}

// sortStrings is a tiny helper to keep fork.go dependency-light.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
