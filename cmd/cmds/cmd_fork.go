package cmds

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewForkCommand() *cobra.Command {
	var (
		branch       string
		branchPrefix string
		agentSource  string
		dryRun       bool
		workspace    string
	)

	cmd := &cobra.Command{
		Use:   "fork <new-workspace-name> [source-workspace-name]",
		Short: "Create a new workspace by forking an existing workspace",
		Long: `Create a new workspace that is a fork of an existing workspace.
The new workspace will contain the same repositories as the source workspace,
with a new branch created from the current branch of the source workspace.

If no source workspace is specified, attempts to detect the current workspace.

The source workspace's current branch will be used as the base branch for 
the new workspace's branch.

Examples:
  # Fork current workspace to create "my-feature"
  workspace-manager fork my-feature

  # Fork a specific workspace
  workspace-manager fork my-feature source-workspace

  # Fork with custom branch name
  workspace-manager fork my-feature --branch feature/new-api

  # Fork with custom branch prefix (bug/my-feature)
  workspace-manager fork my-feature --branch-prefix bug`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			newWorkspaceName := args[0]
			sourceWorkspaceName := workspace
			if len(args) > 1 {
				sourceWorkspaceName = args[1]
			}
			return runFork(cmd.Context(), newWorkspaceName, sourceWorkspaceName, branch, branchPrefix, agentSource, dryRun)
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "Branch name for the new workspace (if not specified, uses <branch-prefix>/<new-workspace-name>)")
	cmd.Flags().StringVar(&branchPrefix, "branch-prefix", "task", "Prefix for auto-generated branch names")
	cmd.Flags().StringVar(&agentSource, "agent-source", "", "Path to AGENT.md template file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be created without actually creating")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Source workspace name")

	return cmd
}

func runFork(ctx context.Context, newWorkspaceName, sourceWorkspaceName, branch, branchPrefix, agentSource string, dryRun bool) error {
	workflow, err := workflows.NewForkWorkflow()
	if err != nil {
		return err
	}

	req := workflows.ForkRequest{
		NewWorkspaceName:    newWorkspaceName,
		SourceWorkspaceName: sourceWorkspaceName,
		Branch:              branch,
		BranchPrefix:        branchPrefix,
		AgentSource:         agentSource,
		DryRun:              dryRun,
	}

	plan, err := workflow.Plan(ctx, req)
	if err != nil {
		return err
	}

	sourceWorkspace := plan.SourceWorkspace
	output.PrintInfo("Forking workspace '%s' to create '%s'", sourceWorkspace.Name, newWorkspaceName)
	output.PrintInfo("Using base branch: %s", plan.BaseBranch)
	if branch == "" {
		output.PrintInfo("Using auto-generated branch: %s", plan.FinalBranch)
		log.Debug().Str("branch", plan.FinalBranch).Str("prefix", branchPrefix).Str("name", newWorkspaceName).Msg("Generated branch name")
	}
	if agentSource == "" && plan.FinalAgentSource != "" {
		output.PrintInfo("Using AGENT.md from source workspace: %s", plan.FinalAgentSource)
	}

	// Create the new workspace
	log.Debug().
		Str("newName", newWorkspaceName).
		Str("sourceName", sourceWorkspace.Name).
		Strs("repos", plan.RepoNames).
		Str("branch", plan.FinalBranch).
		Str("baseBranch", plan.BaseBranch).
		Bool("dryRun", dryRun).
		Msg("Forking workspace")

	workspace, _, err := workflow.Fork(ctx, req)
	if err != nil {
		// Check if user cancelled - handle gracefully without error
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "cancelled by user") ||
			strings.Contains(errMsg, "creation cancelled") ||
			strings.Contains(errMsg, "operation cancelled") {
			output.PrintInfo("Operation cancelled.")
			return nil // Return success to prevent usage help
		}
		return err
	}

	// Show results
	if dryRun {
		output.PrintHeader("📋 Fork Preview: %s → %s", sourceWorkspace.Name, workspace.Name)
		fmt.Println()
		output.PrintInfo("Source workspace:")
		fmt.Printf("  Name: %s\n", sourceWorkspace.Name)
		fmt.Printf("  Path: %s\n", sourceWorkspace.Path)
		fmt.Printf("  Current branch: %s\n", plan.BaseBranch)
		fmt.Println()
		return showWorkspacePreview(workspace)
	}

	output.PrintSuccess("Workspace '%s' forked successfully from '%s'!", workspace.Name, sourceWorkspace.Name)
	fmt.Println()

	output.PrintHeader("Fork Details")
	fmt.Printf("  Source: %s (branch: %s)\n", sourceWorkspace.Name, plan.BaseBranch)
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

	return nil
}
