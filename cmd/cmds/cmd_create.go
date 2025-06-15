package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	var (
		repos        []string
		branch       string
		branchPrefix string
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
  workspace-manager create my-feature --repos app,lib --branch-prefix bug`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.Context(), args[0], repos, branch, branchPrefix, agentSource, interactive, dryRun)
		},
	}

	cmd.Flags().StringSliceVar(&repos, "repos", nil, "Repository names to include (comma-separated)")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch name for worktrees (if not specified, uses <branch-prefix>/<workspace-name>)")
	cmd.Flags().StringVar(&branchPrefix, "branch-prefix", "task", "Prefix for auto-generated branch names")
	cmd.Flags().StringVar(&agentSource, "agent-source", "", "Path to AGENT.md template file")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive repository selection")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be created without actually creating")

	return cmd
}

func runCreate(ctx context.Context, name string, repos []string, branch, branchPrefix, agentSource string, interactive, dryRun bool) error {
	wm, err := wsm.NewWorkspaceManager()
	if err != nil {
		return errors.Wrap(err, "failed to create workspace manager")
	}

	// Handle interactive mode
	if interactive {
		selectedRepos, err := selectRepositoriesInteractively(wm)
		if err != nil {
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
		fmt.Printf("No branch specified, using auto-generated branch: %s\n", finalBranch)
	}

	// Create workspace
	workspace, err := wm.CreateWorkspace(ctx, name, repos, finalBranch, agentSource, dryRun)
	if err != nil {
		return errors.Wrap(err, "failed to create workspace")
	}

	// Show results
	if dryRun {
		return showWorkspacePreview(workspace)
	}

	fmt.Printf("✅ Workspace '%s' created successfully!\n\n", workspace.Name)
	fmt.Printf("Path: %s\n", workspace.Path)
	fmt.Printf("Repositories: %s\n", strings.Join(getRepositoryNames(workspace.Repositories), ", "))
	if workspace.Branch != "" {
		fmt.Printf("Branch: %s\n", workspace.Branch)
	}
	if workspace.GoWorkspace {
		fmt.Printf("Go workspace: yes (go.work created)\n")
	}
	if workspace.AgentMD != "" {
		fmt.Printf("AGENT.md: copied from %s\n", workspace.AgentMD)
	}

	fmt.Printf("\nTo start working:\n")
	fmt.Printf("  cd %s\n", workspace.Path)

	return nil
}

func selectRepositoriesInteractively(wm *wsm.WorkspaceManager) ([]string, error) {
	repos := wm.Discoverer.GetRepositories()

	if len(repos) == 0 {
		return nil, errors.New("no repositories found. Run 'workspace-manager discover' first")
	}

	fmt.Println("Available repositories:")
	for i, repo := range repos {
		fmt.Printf("  %d. %s (%s) [%s]\n", i+1, repo.Name, repo.Path, strings.Join(repo.Categories, ", "))
	}

	fmt.Print("\nEnter repository numbers (comma-separated) or names: ")
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return nil, errors.Wrap(err, "failed to read input")
	}

	// Parse input - could be numbers or names
	parts := strings.Split(input, ",")
	var selected []string

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Try as number first
		var found bool
		for i, repo := range repos {
			if part == fmt.Sprintf("%d", i+1) {
				selected = append(selected, repo.Name)
				found = true
				break
			}
		}

		// If not found as number, try as name
		if !found {
			for _, repo := range repos {
				if repo.Name == part {
					selected = append(selected, repo.Name)
					found = true
					break
				}
			}
		}

		if !found {
			return nil, errors.Errorf("repository not found: %s", part)
		}
	}

	return selected, nil
}

func showWorkspacePreview(workspace *wsm.Workspace) error {
	fmt.Printf("📋 Workspace Preview: %s\n\n", workspace.Name)

	fmt.Printf("Actions to be performed:\n")
	fmt.Printf("1. Create directory structure at: %s\n", workspace.Path)

	fmt.Printf("2. Create worktrees:\n")
	for _, repo := range workspace.Repositories {
		if workspace.Branch != "" {
			fmt.Printf("   git worktree add -B %s %s/%s\n", workspace.Branch, workspace.Path, repo.Name)
		} else {
			fmt.Printf("   git worktree add %s/%s\n", workspace.Path, repo.Name)
		}
	}

	if workspace.GoWorkspace {
		fmt.Printf("3. Initialize go.work and add modules\n")
	}

	if workspace.AgentMD != "" {
		fmt.Printf("4. Copy AGENT.md from %s\n", workspace.AgentMD)
	}

	fmt.Printf("\nRepositories to include:\n")
	for _, repo := range workspace.Repositories {
		fmt.Printf("  - %s (%s) [%s]\n", repo.Name, repo.Path, strings.Join(repo.Categories, ", "))
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
