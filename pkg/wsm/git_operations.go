package wsm

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
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
	Status     string `json:"status"` // M, A, D, R, ?, etc.
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
	gc, _ := BuildGitBackends(ctx)

	for _, repo := range gops.workspace.Repositories {
		repoPath := filepath.Join(gops.workspace.Path, repo.Name)
		h, err := gc.Open(ctx, repoPath)
		if err != nil {
			return nil, errors.Wrapf(err, "open repo %s", repo.Name)
		}
		st, err := gc.Status(ctx, h)
		if err != nil {
			return nil, errors.Wrapf(err, "status for %s", repo.Name)
		}

		var repoChanges []FileChange
		for _, p := range st.StagedFiles {
			repoChanges = append(repoChanges, FileChange{Repository: repo.Name, FilePath: p, Status: "M", Staged: true})
		}
		for _, p := range st.ModifiedFiles {
			repoChanges = append(repoChanges, FileChange{Repository: repo.Name, FilePath: p, Status: "M", Staged: false})
		}
		for _, p := range st.UntrackedFiles {
			repoChanges = append(repoChanges, FileChange{Repository: repo.Name, FilePath: p, Status: "?", Staged: false})
		}

		if len(repoChanges) > 0 {
			changes[repo.Name] = repoChanges
		}
	}

	return changes, nil
}

// StageFile stages a specific file in a repository
func (gops *GitOperations) StageFile(ctx context.Context, repoName, filePath string) error {
	repoPath := filepath.Join(gops.workspace.Path, repoName)
	gc, _ := BuildGitBackends(ctx)
	h, err := gc.Open(ctx, repoPath)
	if err != nil {
		return errors.Wrap(err, "open repo")
	}
	if err := gc.Add(ctx, h, filePath); err != nil {
		return errors.Wrapf(err, "failed to stage file %s in %s", filePath, repoName)
	}

	output.LogInfo(
		fmt.Sprintf("Staged file %s in %s", filePath, repoName),
		"File staged",
		"repository", repoName,
		"file", filePath,
	)

	return nil
}

// UnstageFile unstages a specific file in a repository
func (gops *GitOperations) UnstageFile(ctx context.Context, repoName, filePath string) error {
	repoPath := filepath.Join(gops.workspace.Path, repoName)
	gc, _ := BuildGitBackends(ctx)
	h, err := gc.Open(ctx, repoPath)
	if err != nil {
		return errors.Wrap(err, "open repo")
	}
	if err := gc.Reset(ctx, h, filePath); err != nil {
		return errors.Wrapf(err, "failed to unstage file %s in %s", filePath, repoName)
	}

	output.LogInfo(
		fmt.Sprintf("Unstaged file %s in %s", filePath, repoName),
		"File unstaged",
		"repository", repoName,
		"file", filePath,
	)

	return nil
}

// CommitChanges commits changes across repositories
func (gops *GitOperations) CommitChanges(ctx context.Context, operation *CommitOperation) error {
	if operation.DryRun {
		return gops.previewCommit(ctx, operation)
	}

	gc, _ := BuildGitBackends(ctx)
	var errs []string
	var successfulRepos []string

	for repoName, files := range operation.Files {
		repoPath := filepath.Join(gops.workspace.Path, repoName)
		h, err := gc.Open(ctx, repoPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", repoName, err))
			continue
		}

		// Stage files if needed
		if operation.AddAll {
			if err := gc.Add(ctx, h, "."); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", repoName, err))
				continue
			}
		} else {
			for _, file := range files {
				if !file.Staged {
					if err := gc.Add(ctx, h, file.FilePath); err != nil {
						errs = append(errs, fmt.Sprintf("%s: %v", repoName, err))
						continue
					}
				}
			}
		}

		// Check if there are staged changes
		if st, err := gc.Status(ctx, h); err != nil {
			errs = append(errs, fmt.Sprintf("%s: failed to check staged changes: %v", repoName, err))
			continue
		} else if len(st.StagedFiles) == 0 {
			output.LogInfo(
				fmt.Sprintf("No staged changes in %s, skipping commit", repoName),
				"No staged changes, skipping commit",
				"repository", repoName,
			)
			continue
		}

		// Commit changes
		if _, err := gc.Commit(ctx, h, operation.Message, gitclient.CommitOptions{}); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", repoName, err))
			continue
		}

		successfulRepos = append(successfulRepos, repoName)
	}

	// Push changes if requested
	if operation.Push && len(successfulRepos) > 0 {
		for _, repoName := range successfulRepos {
			repoPath := filepath.Join(gops.workspace.Path, repoName)
			h, err := gc.Open(ctx, repoPath)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s push: %v", repoName, err))
				continue
			}
			if err := gc.Push(ctx, h, ""); err != nil {
				errs = append(errs, fmt.Sprintf("%s push: %v", repoName, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("commit failed for some repositories:\n%s", strings.Join(errs, "\n"))
	}

	output.LogInfo(
		fmt.Sprintf("Successfully committed to %d repositories", len(successfulRepos)),
		"Commit operation completed successfully",
		"repositories", successfulRepos,
		"message", operation.Message,
		"pushed", operation.Push,
	)

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

// GetDiff gets unified diff across repositories
func (gops *GitOperations) GetDiff(ctx context.Context, staged bool, repoFilter string) (string, error) {
	var allDiffs []string
	gc, _ := BuildGitBackends(ctx)

	for _, repo := range gops.workspace.Repositories {
		if repoFilter != "" && repo.Name != repoFilter {
			continue
		}

		repoPath := filepath.Join(gops.workspace.Path, repo.Name)
		h, err := gc.Open(ctx, repoPath)
		if err != nil {
			return "", errors.Wrapf(err, "open repo %s", repo.Name)
		}
		diff, err := gc.Diff(ctx, h, staged)
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

// DiffOptions configures diff collection behavior.
type DiffOptions struct {
	MaxJobs int
}

// GetDiffWithOptions gets unified diff across repositories, with concurrency options.
func (gops *GitOperations) GetDiffWithOptions(ctx context.Context, staged bool, repoFilter string, opts DiffOptions) (string, error) {
	maxJobs := opts.MaxJobs
	if maxJobs < 1 { maxJobs = 1 }

	repos := make([]Repository, 0, len(gops.workspace.Repositories))
	for _, r := range gops.workspace.Repositories {
		if repoFilter != "" && r.Name != repoFilter { continue }
		repos = append(repos, r)
	}

	if len(repos) == 0 {
		return "No changes found in workspace.", nil
	}

	gc, _ := BuildGitBackends(ctx)
	parts := make([]string, 0, len(repos)*2)

	if maxJobs == 1 || len(repos) <= 1 {
		for _, repo := range repos {
			repoPath := filepath.Join(gops.workspace.Path, repo.Name)
			h, err := gc.Open(ctx, repoPath)
			if err != nil { return "", errors.Wrapf(err, "open repo %s", repo.Name) }
			d, err := gc.Diff(ctx, h, staged)
			if err != nil { return "", errors.Wrapf(err, "failed to get diff for %s", repo.Name) }
			if d != "" {
				parts = append(parts, fmt.Sprintf("=== Repository: %s ===", repo.Name), d)
			}
		}
	} else {
		sem := semaphore.NewWeighted(int64(maxJobs))
		g, gctx := errgroup.WithContext(ctx)
		results := make([]struct{ header string; body string }, len(repos))
		for i := range repos {
			i := i
			if err := sem.Acquire(gctx, 1); err != nil { return "", err }
			g.Go(func() error {
				defer sem.Release(1)
				repo := repos[i]
				repoPath := filepath.Join(gops.workspace.Path, repo.Name)
				h, err := gc.Open(gctx, repoPath)
				if err != nil { return errors.Wrapf(err, "open repo %s", repo.Name) }
				d, err := gc.Diff(gctx, h, staged)
				if err != nil { return errors.Wrapf(err, "failed to get diff for %s", repo.Name) }
				if d != "" {
					results[i].header = fmt.Sprintf("=== Repository: %s ===", repo.Name)
					results[i].body = d
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil { return "", err }
		for _, r := range results {
			if r.body != "" {
				parts = append(parts, r.header, r.body)
			}
		}
	}

	if len(parts) == 0 {
		return "No changes found in workspace.", nil
	}
	return strings.Join(parts, "\n"), nil
}
