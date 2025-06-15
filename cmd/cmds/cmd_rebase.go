package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// NewRebaseCommand creates the rebase command
func NewRebaseCommand() *cobra.Command {
	var (
		targetBranch string
		repository   string
		dryRun       bool
		interactive  bool
	)

	cmd := &cobra.Command{
		Use:   "rebase [repository]",
		Short: "Rebase workspace repositories",
		Long: `Rebase workspace repositories against a target branch.

By default, rebases all repositories in the workspace against the 'main' branch.
You can specify a specific repository to rebase or change the target branch.

Examples:
  # Rebase all repositories against main
  workspace-manager rebase

  # Rebase specific repository against main  
  workspace-manager rebase my-repo

  # Rebase all repositories against develop
  workspace-manager rebase --target develop

  # Rebase specific repository against feature/base
  workspace-manager rebase my-repo --target feature/base

  # Interactive rebase
  workspace-manager rebase my-repo --interactive

  # Dry run to see what would be done
  workspace-manager rebase --dry-run`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				repository = args[0]
			}
			return runRebase(cmd.Context(), repository, targetBranch, interactive, dryRun)
		},
	}

	cmd.Flags().StringVar(&targetBranch, "target", "main", "Target branch to rebase onto")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without actually rebasing")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive rebase")

	return cmd
}

// RebaseResult represents the result of a rebase operation
type RebaseResult struct {
	Repository    string `json:"repository"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
	Rebased       bool   `json:"rebased"`
	Conflicts     bool   `json:"conflicts"`
	CommitsBefore int    `json:"commits_before"`
	CommitsAfter  int    `json:"commits_after"`
	TargetBranch  string `json:"target_branch"`
}

func runRebase(ctx context.Context, repository, targetBranch string, interactive, dryRun bool) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	if repository != "" {
		fmt.Printf("🔄 Rebasing repository '%s' onto '%s'\n", repository, targetBranch)
	} else {
		fmt.Printf("🔄 Rebasing all repositories onto '%s'\n", targetBranch)
	}

	if dryRun {
		fmt.Println("📋 Dry run mode - no changes will be made")
	}

	var results []RebaseResult

	if repository != "" {
		// Rebase specific repository
		result := rebaseRepository(ctx, workspace, repository, targetBranch, interactive, dryRun)
		results = append(results, result)
	} else {
		// Rebase all repositories
		for _, repo := range workspace.Repositories {
			result := rebaseRepository(ctx, workspace, repo.Name, targetBranch, interactive, dryRun)
			results = append(results, result)
		}
	}

	return printRebaseResults(results, dryRun)
}

func rebaseRepository(ctx context.Context, workspace *wsm.Workspace, repoName, targetBranch string, interactive, dryRun bool) RebaseResult {
	result := RebaseResult{
		Repository:   repoName,
		Success:      true,
		TargetBranch: targetBranch,
	}

	repoPath := filepath.Join(workspace.Path, repoName)

	// Check if repository exists in workspace
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		result.Success = false
		result.Error = "repository not found in workspace"
		return result
	}

	// Get current branch
	currentBranch, err := getCurrentBranch(ctx, repoPath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to get current branch: %v", err)
		return result
	}

	// Check if we're already on the target branch
	if currentBranch == targetBranch {
		result.Success = true
		result.Error = fmt.Sprintf("already on target branch '%s'", targetBranch)
		return result
	}

	// Get commits count before rebase
	commitsBefore, err := getCommitsAhead(ctx, repoPath, targetBranch)
	if err != nil {
		log.Warn().Err(err).Str("repo", repoName).Msg("Could not get commits count before rebase")
	}
	result.CommitsBefore = commitsBefore

	if dryRun {
		result.Error = "dry-run mode"
		return result
	}

	// Check if target branch exists
	if !branchExists(ctx, repoPath, targetBranch) {
		// Try to fetch it from remote
		if err := fetchBranch(ctx, repoPath, targetBranch); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("target branch '%s' not found locally or on remote", targetBranch)
			return result
		}
	}

	// Perform rebase
	if err := performRebase(ctx, repoPath, targetBranch, interactive); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("rebase failed: %v", err)
		result.Conflicts = hasRebaseConflicts(ctx, repoPath)
		return result
	}

	result.Rebased = true

	// Get commits count after rebase
	commitsAfter, err := getCommitsAhead(ctx, repoPath, targetBranch)
	if err != nil {
		log.Warn().Err(err).Str("repo", repoName).Msg("Could not get commits count after rebase")
	}
	result.CommitsAfter = commitsAfter

	log.Info().
		Str("repository", repoName).
		Str("target", targetBranch).
		Int("commits_before", result.CommitsBefore).
		Int("commits_after", result.CommitsAfter).
		Msg("Repository rebase completed")

	return result
}

func getCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func branchExists(ctx context.Context, repoPath, branch string) bool {
	cmd := exec.CommandContext(ctx, "git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = repoPath
	return cmd.Run() == nil
}

func fetchBranch(ctx context.Context, repoPath, branch string) error {
	// Try to fetch the branch from origin
	cmd := exec.CommandContext(ctx, "git", "fetch", "origin", branch+":"+branch)
	cmd.Dir = repoPath
	return cmd.Run()
}

func performRebase(ctx context.Context, repoPath, targetBranch string, interactive bool) error {
	var cmd *exec.Cmd
	if interactive {
		cmd = exec.CommandContext(ctx, "git", "rebase", "-i", targetBranch)
	} else {
		cmd = exec.CommandContext(ctx, "git", "rebase", targetBranch)
	}
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "git rebase failed: %s", string(output))
	}

	return nil
}

func getCommitsAhead(ctx context.Context, repoPath, targetBranch string) (int, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("HEAD..%s", targetBranch))
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count); err != nil {
		return 0, err
	}

	return count, nil
}

func hasRebaseConflicts(ctx context.Context, repoPath string) bool {
	// Check if rebase is in progress
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) >= 2 && (line[0] == 'U' || line[1] == 'U' ||
			(line[0] == 'A' && line[1] == 'A') ||
			(line[0] == 'D' && line[1] == 'D')) {
			return true
		}
	}

	// Also check if .git/rebase-merge or .git/rebase-apply exists
	rebaseMergeDir := filepath.Join(repoPath, ".git", "rebase-merge")
	rebaseApplyDir := filepath.Join(repoPath, ".git", "rebase-apply")

	if _, err := os.Stat(rebaseMergeDir); err == nil {
		return true
	}
	if _, err := os.Stat(rebaseApplyDir); err == nil {
		return true
	}

	return false
}

func printRebaseResults(results []RebaseResult, dryRun bool) error {
	if len(results) == 0 {
		fmt.Println("No repositories to rebase.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "\nREPOSITORY\tSTATUS\tTARGET\tCOMMITS BEFORE\tCOMMITS AFTER\tERROR")
	fmt.Fprintln(w, "----------\t------\t------\t--------------\t-------------\t-----")

	successCount := 0
	conflictCount := 0

	for _, result := range results {
		status := "✅"
		if !result.Success {
			status = "❌"
		} else {
			successCount++
		}

		if result.Conflicts {
			status = "⚠️"
			conflictCount++
		}

		commitsBefore := "-"
		if result.CommitsBefore > 0 {
			commitsBefore = fmt.Sprintf("%d", result.CommitsBefore)
		}

		commitsAfter := "-"
		if result.CommitsAfter > 0 {
			commitsAfter = fmt.Sprintf("%d", result.CommitsAfter)
		}

		errorMsg := result.Error
		if len(errorMsg) > 30 {
			errorMsg = errorMsg[:27] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			result.Repository,
			status,
			result.TargetBranch,
			commitsBefore,
			commitsAfter,
			errorMsg,
		)
	}

	fmt.Fprintln(w)

	// Summary
	fmt.Printf("Summary: %d/%d repositories rebased successfully", successCount, len(results))
	if conflictCount > 0 {
		fmt.Printf(", %d with conflicts", conflictCount)
	}
	fmt.Println()

	if conflictCount > 0 {
		fmt.Println("\n⚠️  Some repositories have conflicts. Resolve them manually with:")
		fmt.Println("  - Fix conflicts in the affected files")
		fmt.Println("  - git add <resolved-files>")
		fmt.Println("  - git rebase --continue")
		fmt.Println("  Or abort the rebase with: git rebase --abort")
	}

	return nil
}
