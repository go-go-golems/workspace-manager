package wsm

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// GitOperations handles git operations across workspace repositories
type GitOperations struct {
	workspace *Workspace
}

// NewGitOperations creates a new git operations handler
func NewGitOperations(workspace *Workspace) *GitOperations {
	return &GitOperations{
		workspace: workspace,
	}
}

// FileChange represents a change to a file
type FileChange struct {
	Repository string `json:"repository"`
	FilePath   string `json:"file_path"`
	Status     string `json:"status"` // M, A, D, R, etc.
	Staged     bool   `json:"staged"`
}

// CommitOperation represents a commit operation across repositories
type CommitOperation struct {
	Message string                  `json:"message"`
	Files   map[string][]FileChange `json:"files"` // repo -> files
	DryRun  bool                    `json:"dry_run"`
	AddAll  bool                    `json:"add_all"`
	Push    bool                    `json:"push"`
}

// GetWorkspaceChanges gets all changes across workspace repositories
func (gops *GitOperations) GetWorkspaceChanges(ctx context.Context) (map[string][]FileChange, error) {
	changes := make(map[string][]FileChange)

	for _, repo := range gops.workspace.Repositories {
		repoPath := filepath.Join(gops.workspace.Path, repo.Name)
		repoChanges, err := gops.getRepositoryChanges(ctx, repo.Name, repoPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get changes for repository %s", repo.Name)
		}
		if len(repoChanges) > 0 {
			changes[repo.Name] = repoChanges
		}
	}

	return changes, nil
}

// getRepositoryChanges gets changes for a single repository
func (gops *GitOperations) getRepositoryChanges(ctx context.Context, repoName, repoPath string) ([]FileChange, error) {
	// Get git status --porcelain
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get git status for %s", repoName)
	}

	var changes []FileChange
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		if len(line) < 3 {
			continue
		}

		indexStatus := line[0]
		workTreeStatus := line[1]
		filePath := strings.TrimSpace(line[2:])

		// Handle staged changes
		if indexStatus != ' ' && indexStatus != '?' {
			changes = append(changes, FileChange{
				Repository: repoName,
				FilePath:   filePath,
				Status:     string(indexStatus),
				Staged:     true,
			})
		}

		// Handle unstaged changes
		if workTreeStatus != ' ' && workTreeStatus != '?' {
			changes = append(changes, FileChange{
				Repository: repoName,
				FilePath:   filePath,
				Status:     string(workTreeStatus),
				Staged:     false,
			})
		}

		// Handle untracked files
		if indexStatus == '?' && workTreeStatus == '?' {
			changes = append(changes, FileChange{
				Repository: repoName,
				FilePath:   filePath,
				Status:     "?",
				Staged:     false,
			})
		}
	}

	return changes, nil
}

// StageFile stages a specific file in a repository
func (gops *GitOperations) StageFile(ctx context.Context, repoName, filePath string) error {
	repoPath := filepath.Join(gops.workspace.Path, repoName)

	cmd := exec.CommandContext(ctx, "git", "add", filePath)
	cmd.Dir = repoPath

	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to stage file %s in %s: %s", filePath, repoName, string(output))
	}

	log.Info().
		Str("repository", repoName).
		Str("file", filePath).
		Msg("File staged")

	return nil
}

// UnstageFile unstages a specific file in a repository
func (gops *GitOperations) UnstageFile(ctx context.Context, repoName, filePath string) error {
	repoPath := filepath.Join(gops.workspace.Path, repoName)

	cmd := exec.CommandContext(ctx, "git", "reset", "HEAD", filePath)
	cmd.Dir = repoPath

	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to unstage file %s in %s: %s", filePath, repoName, string(output))
	}

	log.Info().
		Str("repository", repoName).
		Str("file", filePath).
		Msg("File unstaged")

	return nil
}

// CommitChanges commits changes across repositories
func (gops *GitOperations) CommitChanges(ctx context.Context, operation *CommitOperation) error {
	if operation.DryRun {
		return gops.previewCommit(ctx, operation)
	}

	var errors []string
	var successfulRepos []string

	for repoName, files := range operation.Files {
		repoPath := filepath.Join(gops.workspace.Path, repoName)

		// Stage files if needed
		if operation.AddAll {
			if err := gops.stageAllFiles(ctx, repoName, repoPath); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", repoName, err))
				continue
			}
		} else {
			// Stage only selected files
			for _, file := range files {
				if !file.Staged {
					if err := gops.StageFile(ctx, repoName, file.FilePath); err != nil {
						errors = append(errors, fmt.Sprintf("%s: %v", repoName, err))
						continue
					}
				}
			}
		}

		// Check if there are staged changes
		if hasStaged, err := gops.hasStagedChanges(ctx, repoPath); err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to check staged changes: %v", repoName, err))
			continue
		} else if !hasStaged {
			log.Info().Str("repository", repoName).Msg("No staged changes, skipping commit")
			continue
		}

		// Commit changes
		if err := gops.commitRepository(ctx, repoName, repoPath, operation.Message); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", repoName, err))
			continue
		}

		successfulRepos = append(successfulRepos, repoName)
	}

	// Push changes if requested
	if operation.Push && len(successfulRepos) > 0 {
		for _, repoName := range successfulRepos {
			repoPath := filepath.Join(gops.workspace.Path, repoName)
			if err := gops.pushRepository(ctx, repoName, repoPath); err != nil {
				errors = append(errors, fmt.Sprintf("%s push: %v", repoName, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("commit failed for some repositories:\n%s", strings.Join(errors, "\n"))
	}

	log.Info().
		Strs("repositories", successfulRepos).
		Str("message", operation.Message).
		Bool("pushed", operation.Push).
		Msg("Commit operation completed successfully")

	return nil
}

// previewCommit shows what would be committed
func (gops *GitOperations) previewCommit(ctx context.Context, operation *CommitOperation) error {
	fmt.Printf("Commit Preview:\n")
	fmt.Printf("Message: %s\n\n", operation.Message)

	for repoName, files := range operation.Files {
		fmt.Printf("Repository: %s\n", repoName)
		for _, file := range files {
			status := "+"
			if file.Staged {
				status = "✓"
			}
			fmt.Printf("  %s %s (%s)\n", status, file.FilePath, file.Status)
		}
		fmt.Println()
	}

	if operation.Push {
		fmt.Println("Changes will be pushed after commit.")
	}

	return nil
}

// stageAllFiles stages all changes in a repository
func (gops *GitOperations) stageAllFiles(ctx context.Context, repoName, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = repoPath

	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to stage all files in %s: %s", repoName, string(output))
	}

	return nil
}

// hasStagedChanges checks if repository has staged changes
func (gops *GitOperations) hasStagedChanges(ctx context.Context, repoPath string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	cmd.Dir = repoPath

	err := cmd.Run()
	if err != nil {
		// Exit code 1 means there are differences (staged changes)
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return true, nil
		}
		return false, err
	}

	// Exit code 0 means no differences (no staged changes)
	return false, nil
}

// commitRepository commits changes in a single repository
func (gops *GitOperations) commitRepository(ctx context.Context, repoName, repoPath, message string) error {
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to commit in %s: %s", repoName, string(output))
	}

	log.Info().
		Str("repository", repoName).
		Str("message", message).
		Msg("Repository committed successfully")

	return nil
}

// pushRepository pushes changes in a single repository
func (gops *GitOperations) pushRepository(ctx context.Context, repoName, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "push")
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to push %s: %s", repoName, string(output))
	}

	log.Info().
		Str("repository", repoName).
		Msg("Repository pushed successfully")

	return nil
}

// GetDiff gets unified diff across repositories
func (gops *GitOperations) GetDiff(ctx context.Context, staged bool, repoFilter string) (string, error) {
	var allDiffs []string

	for _, repo := range gops.workspace.Repositories {
		if repoFilter != "" && repo.Name != repoFilter {
			continue
		}

		repoPath := filepath.Join(gops.workspace.Path, repo.Name)
		diff, err := gops.getRepositoryDiff(ctx, repo.Name, repoPath, staged)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get diff for %s", repo.Name)
		}

		if diff != "" {
			header := fmt.Sprintf("=== Repository: %s ===", repo.Name)
			allDiffs = append(allDiffs, header, diff)
		}
	}

	if len(allDiffs) == 0 {
		return "No changes found in workspace.", nil
	}

	return strings.Join(allDiffs, "\n"), nil
}

// getRepositoryDiff gets diff for a single repository
func (gops *GitOperations) getRepositoryDiff(ctx context.Context, repoName, repoPath string, staged bool) (string, error) {
	var cmd *exec.Cmd
	if staged {
		cmd = exec.CommandContext(ctx, "git", "diff", "--cached")
	} else {
		cmd = exec.CommandContext(ctx, "git", "diff")
	}
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get diff for %s", repoName)
	}

	return string(output), nil
}
