package wsm

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Shared git utility functions used across multiple components

// analyzeRepository extracts metadata from a git repository
func analyzeRepository(ctx context.Context, path string) (*Repository, error) {
	name := filepath.Base(path)

	repo := &Repository{
		Name:        name,
		Path:        path,
		LastUpdated: time.Now(),
		Categories:  categorizeRepository(path),
	}

	// Get remote URL
	if remoteURL, err := getGitRemoteURL(ctx, path); err == nil {
		repo.RemoteURL = remoteURL
	}

	// Get current branch
	if branch, err := getGitCurrentBranch(ctx, path); err == nil {
		repo.CurrentBranch = branch
	}

	// Get all branches
	if branches, err := getGitBranches(ctx, path); err == nil {
		repo.Branches = branches
	}

	// Get tags
	if tags, err := getGitTags(ctx, path); err == nil {
		repo.Tags = tags
	}

	// Get last commit
	if lastCommit, err := getGitLastCommit(ctx, path); err == nil {
		repo.LastCommit = lastCommit
	}

	return repo, nil
}

// categorizeRepository determines categories based on repository content
func categorizeRepository(path string) []string {
	var categories []string

	// Check for common language/framework files
	files := map[string]string{
		"go.mod":             "go",
		"package.json":       "node",
		"Cargo.toml":         "rust",
		"setup.py":           "python",
		"requirements.txt":   "python",
		"Gemfile":            "ruby",
		"pom.xml":            "java",
		"build.gradle":       "gradle",
		"Makefile":           "make",
		"docker-compose.yml": "docker",
		"Dockerfile":         "docker",
	}

	for file, category := range files {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			categories = append(categories, category)
		}
	}

	// Check for common project types
	dirs := map[string]string{
		"cmd":    "cli",
		"web":    "web",
		"mobile": "mobile",
		"tui":    "tui",
		"api":    "api",
		"server": "server",
		"client": "client",
	}

	for dir, category := range dirs {
		if stat, err := os.Stat(filepath.Join(path, dir)); err == nil && stat.IsDir() {
			categories = append(categories, category)
		}
	}

	if len(categories) == 0 {
		categories = append(categories, "unknown")
	}

	return categories
}

// Git command helpers
func getGitRemoteURL(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getGitCurrentBranch(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getGitBranches(ctx context.Context, path string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "-a")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var branches []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove current branch marker and remote prefixes
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "remotes/origin/")
		if !strings.Contains(line, "HEAD ->") {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

func getGitTags(ctx context.Context, path string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "tag", "-l")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var tags []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}

	return tags, nil
}

func getGitLastCommit(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%H %s")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// CheckBranchMerged checks if the current branch has been merged to origin/main
func CheckBranchMerged(ctx context.Context, path string) (bool, error) {
	// Get current branch for logging
	currentBranch, branchErr := getGitCurrentBranch(ctx, path)
	if branchErr != nil {
		log.Debug().Err(branchErr).Str("path", path).Msg("Failed to get current branch for merge check")
		currentBranch = "unknown"
	}

	log.Debug().Str("path", path).Str("branch", currentBranch).Msg("Checking if branch is merged to origin/main")

	// First, fetch to ensure we have latest remote refs
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin", "main")
	fetchCmd.Dir = path
	fetchErr := fetchCmd.Run()
	if fetchErr != nil {
		log.Debug().Err(fetchErr).Str("path", path).Msg("Failed to fetch origin/main - might be offline")
	} else {
		log.Debug().Str("path", path).Msg("Successfully fetched origin/main")
	}

	// Check if current branch is merged into origin/main
	cmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", "HEAD", "origin/main")
	cmd.Dir = path
	err := cmd.Run()

	// If exit code is 0, the branch is merged
	// If exit code is 1, the branch is not merged
	// If exit code is other, there was an error
	if err == nil {
		log.Debug().Str("path", path).Str("branch", currentBranch).Msg("Branch is merged to origin/main")
		return true, nil
	}

	// Check if it's just "not merged" vs actual error
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		log.Debug().Str("path", path).Str("branch", currentBranch).Msg("Branch is NOT merged to origin/main")
		return false, nil
	}

	// Some other error occurred
	log.Debug().Err(err).Str("path", path).Str("branch", currentBranch).Msg("Error checking merge status")
	return false, err
}

// CheckBranchNeedsRebase checks if the current branch needs to be rebased on origin/main
func CheckBranchNeedsRebase(ctx context.Context, path string) (bool, error) {
	// Get current branch for logging
	currentBranch, branchErr := getGitCurrentBranch(ctx, path)
	if branchErr != nil {
		log.Debug().Err(branchErr).Str("path", path).Msg("Failed to get current branch for rebase check")
		currentBranch = "unknown"
	}

	// Skip rebase check if we're on main branch
	if currentBranch == "main" || currentBranch == "master" {
		log.Debug().Str("path", path).Str("branch", currentBranch).Msg("Skipping rebase check - already on main branch")
		return false, nil
	}

	log.Debug().Str("path", path).Str("branch", currentBranch).Msg("Checking if branch needs rebase on origin/main")

	// First, fetch to ensure we have latest remote refs
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin", "main")
	fetchCmd.Dir = path
	fetchErr := fetchCmd.Run()
	if fetchErr != nil {
		log.Debug().Err(fetchErr).Str("path", path).Msg("Failed to fetch origin/main - might be offline")
	} else {
		log.Debug().Str("path", path).Msg("Successfully fetched origin/main")
	}

	// Find the merge-base between current branch and origin/main
	mergeBaseCmd := exec.CommandContext(ctx, "git", "merge-base", "HEAD", "origin/main")
	mergeBaseCmd.Dir = path
	mergeBaseOutput, err := mergeBaseCmd.Output()
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("Failed to find merge-base with origin/main")
		return false, err
	}
	mergeBase := strings.TrimSpace(string(mergeBaseOutput))

	// Check if origin/main has commits that are not in the current branch
	// This is done by checking if origin/main is ahead of the merge-base
	revListCmd := exec.CommandContext(ctx, "git", "rev-list", "--count", mergeBase+"..origin/main")
	revListCmd.Dir = path
	revListOutput, err := revListCmd.Output()
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("Failed to count commits ahead in origin/main")
		return false, err
	}

	commitsAhead := strings.TrimSpace(string(revListOutput))
	needsRebase := commitsAhead != "0"

	log.Debug().
		Str("path", path).
		Str("branch", currentBranch).
		Str("mergeBase", mergeBase).
		Str("commitsAhead", commitsAhead).
		Bool("needsRebase", needsRebase).
		Msg("Branch rebase check completed")

	return needsRebase, nil
}

// mergeRepositories merges existing repositories with newly discovered ones
func mergeRepositories(existing, discovered []Repository) []Repository {
	repoMap := make(map[string]Repository)

	// Add existing repositories
	for _, repo := range existing {
		repoMap[repo.Path] = repo
	}

	// Update with discovered repositories
	for _, repo := range discovered {
		repoMap[repo.Path] = repo
	}

	// Convert back to slice
	var result []Repository
	for _, repo := range repoMap {
		result = append(result, repo)
	}

	return result
}
