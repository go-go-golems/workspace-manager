package cmds

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
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
	wm, err := wsm.NewWorkspaceManager()
	if err != nil {
		return errors.Wrap(err, "failed to create workspace manager")
	}

	// If no source workspace specified, try to detect current workspace
	if sourceWorkspaceName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to get current directory")
		}

		detected, err := detectWorkspace(cwd)
		if err != nil {
			return errors.Wrap(err, "failed to detect workspace. Use 'workspace-manager fork <new-name> <source-workspace>' or specify --workspace flag")
		}
		sourceWorkspaceName = detected
	}

	// Load source workspace
	sourceWorkspace, err := loadWorkspace(sourceWorkspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load source workspace '%s'", sourceWorkspaceName)
	}

	output.PrintInfo("Forking workspace '%s' to create '%s'", sourceWorkspace.Name, newWorkspaceName)

	// Get current branch status of source workspace to use as base branch
	checker := wsm.NewStatusChecker()
	status, err := checker.GetWorkspaceStatus(ctx, sourceWorkspace)
	if err != nil {
		return errors.Wrap(err, "failed to get source workspace status")
	}

	// Determine the base branch from the source workspace
	// Use the first repository's current branch as the base
	var baseBranch string
	if len(status.Repositories) > 0 {
		baseBranch = status.Repositories[0].CurrentBranch
		output.PrintInfo("Using base branch: %s", baseBranch)
	}

	// Validate that all repositories are on the same branch
	for _, repoStatus := range status.Repositories {
		if repoStatus.CurrentBranch != baseBranch {
			return errors.Errorf("repositories in source workspace are on different branches: %s is on %s, but expected %s",
				repoStatus.Repository.Name, repoStatus.CurrentBranch, baseBranch)
		}
	}

	// Generate branch name if not specified
	finalBranch := branch
	if finalBranch == "" {
		finalBranch = fmt.Sprintf("%s/%s", branchPrefix, newWorkspaceName)
		output.PrintInfo("Using auto-generated branch: %s", finalBranch)
		log.Debug().Str("branch", finalBranch).Str("prefix", branchPrefix).Str("name", newWorkspaceName).Msg("Generated branch name")
	}

	// Extract repository names from source workspace
	var repoNames []string
	for _, repo := range sourceWorkspace.Repositories {
		repoNames = append(repoNames, repo.Name)
	}

	// Use the source workspace's agent MD if no custom one specified
	finalAgentSource := agentSource
	if finalAgentSource == "" && sourceWorkspace.AgentMD != "" {
		finalAgentSource = sourceWorkspace.AgentMD
		output.PrintInfo("Using AGENT.md from source workspace: %s", finalAgentSource)
	}

	// Create the new workspace
	log.Debug().
		Str("newName", newWorkspaceName).
		Str("sourceName", sourceWorkspace.Name).
		Strs("repos", repoNames).
		Str("branch", finalBranch).
		Str("baseBranch", baseBranch).
		Bool("dryRun", dryRun).
		Msg("Forking workspace")

	workspace, err := wm.CreateWorkspace(ctx, newWorkspaceName, repoNames, finalBranch, baseBranch, finalAgentSource, dryRun)
	if err != nil {
		// Check if user cancelled - handle gracefully without error
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "cancelled by user") ||
			strings.Contains(errMsg, "creation cancelled") ||
			strings.Contains(errMsg, "operation cancelled") {
			output.PrintInfo("Operation cancelled.")
			return nil // Return success to prevent usage help
		}
		return errors.Wrap(err, "failed to fork workspace")
	}

	// Show results
	if dryRun {
		output.PrintHeader("📋 Fork Preview: %s → %s", sourceWorkspace.Name, workspace.Name)
		fmt.Println()
		output.PrintInfo("Source workspace:")
		fmt.Printf("  Name: %s\n", sourceWorkspace.Name)
		fmt.Printf("  Path: %s\n", sourceWorkspace.Path)
		fmt.Printf("  Current branch: %s\n", baseBranch)
		fmt.Println()
		return showWorkspacePreview(workspace)
	}

	output.PrintSuccess("Workspace '%s' forked successfully from '%s'!", workspace.Name, sourceWorkspace.Name)
	fmt.Println()

	output.PrintHeader("Fork Details")
	fmt.Printf("  Source: %s (branch: %s)\n", sourceWorkspace.Name, baseBranch)
	fmt.Printf("  New workspace: %s\n", workspace.Name)
	fmt.Printf("  Path: %s\n", workspace.Path)
	fmt.Printf("  Repositories: %s\n", strings.Join(getRepositoryNames(workspace.Repositories), ", "))
	fmt.Printf("  New branch: %s\n", workspace.Branch)
	fmt.Printf("  Base branch: %s\n", workspace.BaseBranch)
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
