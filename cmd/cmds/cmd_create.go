package cmds

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	var (
		repos        []string
		branch       string
		branchPrefix string
		baseBranch   string
		agentSource  string
		interactive  bool
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:   "create [workspace-name]",
		Short: "Create a new multi-repository workspace",
		Long: `Create a new workspace with specified repositories.
The workspace will contain git worktrees for each repository on the specified branch.

If no branch is specified, a branch will be automatically created using the pattern:
  <branch-prefix>/<workspace-name>

Examples:
  # Create workspace with automatic branch (task/my-feature)
  workspace-manager create my-feature --repos app,lib

  # Create workspace with custom branch
  workspace-manager create my-feature --repos app,lib --branch feature/new-api

  # Create workspace with custom branch prefix (bug/my-feature)
  workspace-manager create my-feature --repos app,lib --branch-prefix bug

  # Create workspace from specific base branch
  workspace-manager create my-feature --repos app,lib --base-branch main`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.Context(), args[0], repos, branch, branchPrefix, baseBranch, agentSource, interactive, dryRun)
		},
	}

	cmd.Flags().StringSliceVar(&repos, "repos", nil, "Repository names to include (comma-separated)")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch name for worktrees (if not specified, uses <branch-prefix>/<workspace-name>)")
	cmd.Flags().StringVar(&branchPrefix, "branch-prefix", "task", "Prefix for auto-generated branch names")
	cmd.Flags().StringVar(&baseBranch, "base-branch", "", "Base branch to create new branch from (defaults to current branch)")
	cmd.Flags().StringVar(&agentSource, "agent-source", "", "Path to AGENT.md template file")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive repository selection")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be created without actually creating")

	return cmd
}

func runCreate(ctx context.Context, name string, repos []string, branch, branchPrefix, baseBranch, agentSource string, interactive, dryRun bool) error {
	wm, err := wsm.NewWorkspaceManager()
	if err != nil {
		return errors.Wrap(err, "failed to create workspace manager")
	}

	// Handle interactive mode
	if interactive {
		selectedRepos, err := selectRepositoriesInteractively(wm)
		if err != nil {
			// Check if user cancelled - handle gracefully without error
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "cancelled by user") ||
				strings.Contains(errMsg, "creation cancelled") ||
				strings.Contains(errMsg, "operation cancelled") {
				output.PrintInfo("Operation cancelled.")
				return nil // Return success to prevent usage help
			}
			return errors.Wrap(err, "interactive selection failed")
		}
		repos = selectedRepos
	}

	// Validate inputs
	if len(repos) == 0 {
		return errors.New("no repositories specified. Use --repos flag or --interactive mode")
	}

	// Generate branch name if not specified
	finalBranch := branch
	if finalBranch == "" {
		finalBranch = fmt.Sprintf("%s/%s", branchPrefix, name)
		output.PrintInfo("Using auto-generated branch: %s", finalBranch)
		log.Debug().Str("branch", finalBranch).Str("prefix", branchPrefix).Str("name", name).Msg("Generated branch name")
	}

	// Create workspace
	log.Debug().Str("name", name).Strs("repos", repos).Str("branch", finalBranch).Str("baseBranch", baseBranch).Bool("dryRun", dryRun).Msg("Creating workspace")
	workspace, err := wm.CreateWorkspace(ctx, name, repos, finalBranch, baseBranch, agentSource, dryRun)
	if err != nil {
		// Check if user cancelled - handle gracefully without error
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "cancelled by user") ||
			strings.Contains(errMsg, "creation cancelled") ||
			strings.Contains(errMsg, "operation cancelled") {
			output.PrintInfo("Operation cancelled.")
			return nil // Return success to prevent usage help
		}
		return errors.Wrap(err, "failed to create workspace")
	}

	// Show results
	if dryRun {
		return showWorkspacePreview(workspace)
	}

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

	return nil
}

func selectRepositoriesInteractively(wm *wsm.WorkspaceManager) ([]string, error) {
	repos := wm.Discoverer.GetRepositories()

	if len(repos) == 0 {
		return nil, errors.New("no repositories found. Run 'workspace-manager discover' first")
	}

	output.PrintHeader("Select Repositories")

	// Create options for multi-select
	var options []huh.Option[string]
	for _, repo := range repos {
		label := fmt.Sprintf("%s (%s)", repo.Name, strings.Join(repo.Categories, ", "))
		options = append(options, huh.NewOption(label, repo.Name))
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Choose repositories to include:").
				Options(options...).
				Value(&selected),
		),
	)

	log.Debug().Int("repoCount", len(repos)).Msg("Showing interactive repository selection")
	err := form.Run()
	if err != nil {
		// Check if user cancelled/aborted the form
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "user aborted") ||
			strings.Contains(errMsg, "cancelled") ||
			strings.Contains(errMsg, "aborted") ||
			strings.Contains(errMsg, "interrupt") {
			return nil, errors.New("workspace creation cancelled by user")
		}
		return nil, errors.Wrap(err, "interactive form failed")
	}

	if len(selected) == 0 {
		return nil, errors.New("no repositories selected")
	}

	output.PrintInfo("Selected %d repositories: %s", len(selected), strings.Join(selected, ", "))
	return selected, nil
}

func showWorkspacePreview(workspace *wsm.Workspace) error {
	output.PrintHeader("📋 Workspace Preview: %s", workspace.Name)
	fmt.Println()

	output.PrintInfo("Actions to be performed:")
	fmt.Printf("  1. Create directory structure at: %s\n", workspace.Path)

	fmt.Printf("  2. Create worktrees:\n")
	for _, repo := range workspace.Repositories {
		if workspace.Branch != "" {
			fmt.Printf("     git worktree add -B %s %s/%s\n", workspace.Branch, workspace.Path, repo.Name)
		} else {
			fmt.Printf("     git worktree add %s/%s\n", workspace.Path, repo.Name)
		}
	}

	if workspace.GoWorkspace {
		fmt.Printf("  3. Initialize go.work and add modules\n")
	}

	if workspace.AgentMD != "" {
		fmt.Printf("  4. Copy AGENT.md from %s\n", workspace.AgentMD)
	}

	fmt.Println()
	output.PrintInfo("Repositories to include:")
	for _, repo := range workspace.Repositories {
		fmt.Printf("  • %s (%s) [%s]\n", repo.Name, repo.Path, strings.Join(repo.Categories, ", "))
	}

	return nil
}

func getRepositoryNames(repos []wsm.Repository) []string {
	names := make([]string, len(repos))
	for i, repo := range repos {
		names[i] = repo.Name
	}
	return names
}
