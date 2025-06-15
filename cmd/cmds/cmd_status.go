package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewStatusCommand() *cobra.Command {
	var (
		short     bool
		untracked bool
		workspace string
	)

	cmd := &cobra.Command{
		Use:   "status [workspace-name]",
		Short: "Show workspace status",
		Long: `Show the git status of all repositories in a workspace.
If no workspace name is provided, attempts to detect the current workspace.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := workspace
			if len(args) > 0 {
				workspaceName = args[0]
			}
			return runStatus(cmd.Context(), workspaceName, short, untracked)
		},
	}

	cmd.Flags().BoolVar(&short, "short", false, "Show short status format")
	cmd.Flags().BoolVar(&untracked, "untracked", false, "Include untracked files")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")

	return cmd
}

func runStatus(ctx context.Context, workspaceName string, short, untracked bool) error {
	// If no workspace specified, try to detect current workspace
	if workspaceName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to get current directory")
		}

		detected, err := detectWorkspace(cwd)
		if err != nil {
			return errors.Wrap(err, "failed to detect workspace. Use 'workspace-manager status <workspace-name>' or specify --workspace flag")
		}
		workspaceName = detected
	}

	// Load workspace
	workspace, err := loadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	// Get status
	checker := wsm.NewStatusChecker()
	status, err := checker.GetWorkspaceStatus(ctx, workspace)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace status")
	}

	// Display status
	if short {
		return printStatusShort(status, untracked)
	}

	return printStatusDetailed(status, untracked)
}

func detectWorkspace(cwd string) (string, error) {
	log.Debug().Str("cwd", cwd).Msg("Starting workspace detection")

	// First, try to find a workspace that contains this directory
	workspaces, err := wsm.LoadWorkspaces()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to load workspaces")
		return "", errors.Wrap(err, "failed to load workspaces")
	}

	log.Debug().Int("workspaceCount", len(workspaces)).Msg("Loaded workspaces")

	// Check if current directory is within any workspace path
	for _, workspace := range workspaces {
		log.Debug().
			Str("workspaceName", workspace.Name).
			Str("workspacePath", workspace.Path).
			Msg("Checking workspace")

		// Check if current directory is within or matches workspace path
		if strings.HasPrefix(cwd, workspace.Path) {
			log.Info().
				Str("workspaceName", workspace.Name).
				Str("workspacePath", workspace.Path).
				Str("cwd", cwd).
				Msg("Found workspace containing current directory")
			return workspace.Name, nil
		}

		// Also check if any repository in the workspace matches current directory
		for _, repo := range workspace.Repositories {
			repoWorktreePath := filepath.Join(workspace.Path, repo.Name)
			log.Debug().
				Str("repo", repo.Name).
				Str("repoWorktreePath", repoWorktreePath).
				Msg("Checking repository worktree path")

			if strings.HasPrefix(cwd, repoWorktreePath) {
				log.Info().
					Str("workspaceName", workspace.Name).
					Str("repo", repo.Name).
					Str("repoWorktreePath", repoWorktreePath).
					Str("cwd", cwd).
					Msg("Found workspace via repository worktree path")
				return workspace.Name, nil
			}
		}
	}

	log.Debug().Msg("No workspace found containing current directory, trying heuristic detection")

	// Fallback: Look for workspace configuration file in current directory or parents
	dir := cwd

	for {
		log.Debug().Str("dir", dir).Msg("Checking directory for workspace structure")

		// Check if this directory contains repository worktrees
		entries, err := os.ReadDir(dir)
		if err != nil {
			log.Debug().Err(err).Str("dir", dir).Msg("Failed to read directory")
			return "", err
		}

		// Look for .git files (worktree indicators) and workspace structure
		gitDirs := 0
		var gitRepos []string
		for _, entry := range entries {
			if entry.IsDir() {
				gitFile := filepath.Join(dir, entry.Name(), ".git")
				if stat, err := os.Stat(gitFile); err == nil && stat.Mode().IsRegular() {
					gitDirs++
					gitRepos = append(gitRepos, entry.Name())
				}
			}
		}

		log.Debug().
			Str("dir", dir).
			Int("gitDirs", gitDirs).
			Strs("gitRepos", gitRepos).
			Msg("Found git repositories in directory")

		// If we found multiple git worktrees, this might be a workspace
		if gitDirs >= 2 {
			// Try to find a workspace that matches this path
			dirName := filepath.Base(dir)
			log.Debug().Str("dirName", dirName).Msg("Checking if directory name matches any workspace")

			for _, workspace := range workspaces {
				if workspace.Name == dirName || strings.Contains(workspace.Path, dirName) {
					log.Info().
						Str("workspaceName", workspace.Name).
						Str("dirName", dirName).
						Msg("Found workspace by directory name match")
					return workspace.Name, nil
				}
			}

			// If no exact match, return the directory name as best guess
			log.Debug().Str("dirName", dirName).Msg("Using directory name as workspace name")
			return dirName, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			log.Debug().Msg("Reached filesystem root")
			break // Reached root
		}
		dir = parent
	}

	log.Debug().Msg("No workspace detected")
	return "", errors.New("not in a workspace directory")
}

func loadWorkspace(name string) (*wsm.Workspace, error) {
	workspaces, err := wsm.LoadWorkspaces()
	if err != nil {
		return nil, err
	}

	for _, workspace := range workspaces {
		if workspace.Name == name {
			return &workspace, nil
		}
	}

	return nil, errors.Errorf("workspace not found: %s", name)
}

func printStatusShort(status *wsm.WorkspaceStatus, includeUntracked bool) error {
	fmt.Printf("Workspace: %s (%s)\n", status.Workspace.Name, status.Overall)

	for _, repoStatus := range status.Repositories {
		symbol := getRepositoryStatusSymbol(repoStatus)
		fmt.Printf("%s %s", symbol, repoStatus.Repository.Name)

		if repoStatus.CurrentBranch != "" {
			fmt.Printf(" [%s]", repoStatus.CurrentBranch)
		}

		if repoStatus.Ahead > 0 || repoStatus.Behind > 0 {
			fmt.Printf(" ↑%d ↓%d", repoStatus.Ahead, repoStatus.Behind)
		}

		changes := []string{}
		if len(repoStatus.StagedFiles) > 0 {
			changes = append(changes, fmt.Sprintf("S:%d", len(repoStatus.StagedFiles)))
		}
		if len(repoStatus.ModifiedFiles) > 0 {
			changes = append(changes, fmt.Sprintf("M:%d", len(repoStatus.ModifiedFiles)))
		}
		if includeUntracked && len(repoStatus.UntrackedFiles) > 0 {
			changes = append(changes, fmt.Sprintf("U:%d", len(repoStatus.UntrackedFiles)))
		}

		if len(changes) > 0 {
			fmt.Printf(" [%s]", strings.Join(changes, " "))
		}

		fmt.Println()
	}

	return nil
}

func printStatusDetailed(status *wsm.WorkspaceStatus, includeUntracked bool) error {
	fmt.Printf("Workspace: %s\n", status.Workspace.Name)
	fmt.Printf("Path: %s\n", status.Workspace.Path)
	fmt.Printf("Overall Status: %s\n\n", status.Overall)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "REPOSITORY\tBRANCH\tSTATUS\tCHANGES\tSYNC\tMERGED\tREBASE")
	fmt.Fprintln(w, "----------\t------\t------\t-------\t----\t------\t------")

	for _, repoStatus := range status.Repositories {
		repoName := repoStatus.Repository.Name
		branch := repoStatus.CurrentBranch
		if branch == "" {
			branch = "-"
		}

		statusStr := getStatusString(repoStatus)
		changesStr := getChangesString(repoStatus, includeUntracked)
		syncStr := getSyncString(repoStatus)
		mergedStr := getMergedString(repoStatus)
		rebaseStr := getRebaseString(repoStatus)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			repoName, branch, statusStr, changesStr, syncStr, mergedStr, rebaseStr)
	}

	fmt.Fprintln(w)

	// Show detailed changes if any
	for _, repoStatus := range status.Repositories {
		if repoStatus.HasChanges || (includeUntracked && len(repoStatus.UntrackedFiles) > 0) {
			fmt.Printf("\n%s:\n", repoStatus.Repository.Name)

			if len(repoStatus.StagedFiles) > 0 {
				fmt.Printf("  Staged files:\n")
				for _, file := range repoStatus.StagedFiles {
					fmt.Printf("    + %s\n", file)
				}
			}

			if len(repoStatus.ModifiedFiles) > 0 {
				fmt.Printf("  Modified files:\n")
				for _, file := range repoStatus.ModifiedFiles {
					fmt.Printf("    M %s\n", file)
				}
			}

			if includeUntracked && len(repoStatus.UntrackedFiles) > 0 {
				fmt.Printf("  Untracked files:\n")
				for _, file := range repoStatus.UntrackedFiles {
					fmt.Printf("    ? %s\n", file)
				}
			}
		}
	}

	return nil
}

func getRepositoryStatusSymbol(status wsm.RepositoryStatus) string {
	if status.HasConflicts {
		return "⚠️ "
	}
	if status.HasChanges {
		return "🔄"
	}
	if status.Ahead > 0 || status.Behind > 0 {
		return "📤"
	}
	return "✅"
}

func getStatusString(status wsm.RepositoryStatus) string {
	if status.HasConflicts {
		return "conflict"
	}
	if status.HasChanges {
		return "modified"
	}
	return "clean"
}

func getChangesString(status wsm.RepositoryStatus, includeUntracked bool) string {
	parts := []string{}

	if len(status.StagedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("S:%d", len(status.StagedFiles)))
	}
	if len(status.ModifiedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("M:%d", len(status.ModifiedFiles)))
	}
	if includeUntracked && len(status.UntrackedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("U:%d", len(status.UntrackedFiles)))
	}

	if len(parts) == 0 {
		return "-"
	}

	return strings.Join(parts, " ")
}

func getSyncString(status wsm.RepositoryStatus) string {
	if status.Ahead == 0 && status.Behind == 0 {
		return "✓"
	}
	return fmt.Sprintf("↑%d ↓%d", status.Ahead, status.Behind)
}

func getMergedString(status wsm.RepositoryStatus) string {
	if status.IsMerged {
		return "✓"
	}
	return "-"
}

func getRebaseString(status wsm.RepositoryStatus) string {
	if status.NeedsRebase {
		return "⚠️"
	}
	return "✓"
}
