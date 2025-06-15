package wsm

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// RepositoryDiscoverer handles repository discovery operations
type RepositoryDiscoverer struct {
	registry     *RepositoryRegistry
	registryPath string
}

// NewRepositoryDiscoverer creates a new repository discoverer
func NewRepositoryDiscoverer(registryPath string) *RepositoryDiscoverer {
	return &RepositoryDiscoverer{
		registry:     &RepositoryRegistry{},
		registryPath: registryPath,
	}
}

// LoadRegistry loads the repository registry from disk
func (rd *RepositoryDiscoverer) LoadRegistry() error {
	if _, err := os.Stat(rd.registryPath); os.IsNotExist(err) {
		// Registry doesn't exist, create empty one
		rd.registry = &RepositoryRegistry{
			Repositories: []Repository{},
			LastScan:     time.Time{},
		}
		return nil
	}

	data, err := os.ReadFile(rd.registryPath)
	if err != nil {
		return errors.Wrap(err, "failed to read registry file")
	}

	if err := json.Unmarshal(data, rd.registry); err != nil {
		return errors.Wrap(err, "failed to parse registry file")
	}

	return nil
}

// SaveRegistry saves the repository registry to disk
func (rd *RepositoryDiscoverer) SaveRegistry() error {
	// Ensure directory exists
	dir := filepath.Dir(rd.registryPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "failed to create registry directory")
	}

	data, err := json.MarshalIndent(rd.registry, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal registry")
	}

	if err := os.WriteFile(rd.registryPath, data, 0644); err != nil {
		return errors.Wrap(err, "failed to write registry file")
	}

	return nil
}

// DiscoverRepositories discovers git repositories in the given paths
func (rd *RepositoryDiscoverer) DiscoverRepositories(ctx context.Context, paths []string, recursive bool, maxDepth int) error {
	log.Info().Msg("Starting repository discovery")

	var allRepos []Repository

	for _, path := range paths {
		repos, err := rd.scanDirectory(ctx, path, recursive, maxDepth, 0)
		if err != nil {
			return errors.Wrapf(err, "failed to scan directory %s", path)
		}
		allRepos = append(allRepos, repos...)
	}

	// Update registry
	rd.registry.Repositories = rd.mergeRepositories(rd.registry.Repositories, allRepos)
	rd.registry.LastScan = time.Now()

	log.Info().Int("count", len(allRepos)).Msg("Discovery completed")

	return rd.SaveRegistry()
}

// scanDirectory recursively scans a directory for git repositories
func (rd *RepositoryDiscoverer) scanDirectory(ctx context.Context, path string, recursive bool, maxDepth, currentDepth int) ([]Repository, error) {
	if currentDepth > maxDepth {
		return nil, nil
	}

	var repos []Repository

	// Check if current directory is a git repository
	if rd.isGitRepository(path) {
		repo, err := rd.analyzeRepository(ctx, path)
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Failed to analyze repository")
		} else {
			repos = append(repos, *repo)
		}
	}

	if !recursive {
		return repos, nil
	}

	// Scan subdirectories
	entries, err := os.ReadDir(path)
	if err != nil {
		return repos, errors.Wrapf(err, "failed to read directory %s", path)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories and common non-code directories
		name := entry.Name()
		if strings.HasPrefix(name, ".") && name != ".git" {
			continue
		}
		if name == "node_modules" || name == "vendor" || name == "target" {
			continue
		}

		subPath := filepath.Join(path, name)
		subRepos, err := rd.scanDirectory(ctx, subPath, recursive, maxDepth, currentDepth+1)
		if err != nil {
			log.Warn().Err(err).Str("path", subPath).Msg("Failed to scan subdirectory")
			continue
		}
		repos = append(repos, subRepos...)
	}

	return repos, nil
}

// isGitRepository checks if a directory is a git repository
func (rd *RepositoryDiscoverer) isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if stat, err := os.Stat(gitDir); err == nil {
		return stat.IsDir() || stat.Mode().IsRegular() // .git can be a file in worktrees
	}
	return false
}

// analyzeRepository extracts metadata from a git repository
func (rd *RepositoryDiscoverer) analyzeRepository(ctx context.Context, path string) (*Repository, error) {
	name := filepath.Base(path)

	repo := &Repository{
		Name:        name,
		Path:        path,
		LastUpdated: time.Now(),
		Categories:  rd.categorizeRepository(path),
	}

	// Get remote URL
	if remoteURL, err := rd.getGitRemoteURL(ctx, path); err == nil {
		repo.RemoteURL = remoteURL
	}

	// Get current branch
	if branch, err := rd.getGitCurrentBranch(ctx, path); err == nil {
		repo.CurrentBranch = branch
	}

	// Get all branches
	if branches, err := rd.getGitBranches(ctx, path); err == nil {
		repo.Branches = branches
	}

	// Get tags
	if tags, err := rd.getGitTags(ctx, path); err == nil {
		repo.Tags = tags
	}

	// Get last commit
	if lastCommit, err := rd.getGitLastCommit(ctx, path); err == nil {
		repo.LastCommit = lastCommit
	}

	return repo, nil
}

// categorizeRepository determines categories based on repository content
func (rd *RepositoryDiscoverer) categorizeRepository(path string) []string {
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
func (rd *RepositoryDiscoverer) getGitRemoteURL(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (rd *RepositoryDiscoverer) getGitCurrentBranch(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (rd *RepositoryDiscoverer) getGitBranches(ctx context.Context, path string) ([]string, error) {
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

func (rd *RepositoryDiscoverer) getGitTags(ctx context.Context, path string) ([]string, error) {
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

func (rd *RepositoryDiscoverer) getGitLastCommit(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%H %s")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// mergeRepositories merges existing repositories with newly discovered ones
func (rd *RepositoryDiscoverer) mergeRepositories(existing, discovered []Repository) []Repository {
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

// GetRepositories returns all discovered repositories
func (rd *RepositoryDiscoverer) GetRepositories() []Repository {
	return rd.registry.Repositories
}

// GetRepositoriesByTags returns repositories filtered by tags
func (rd *RepositoryDiscoverer) GetRepositoriesByTags(tags []string) []Repository {
	if len(tags) == 0 {
		return rd.registry.Repositories
	}

	var result []Repository
	for _, repo := range rd.registry.Repositories {
		if rd.hasAnyTag(repo.Categories, tags) {
			result = append(result, repo)
		}
	}

	return result
}

// hasAnyTag checks if repository has any of the specified tags
func (rd *RepositoryDiscoverer) hasAnyTag(repoTags, filterTags []string) bool {
	for _, filterTag := range filterTags {
		for _, repoTag := range repoTags {
			if repoTag == filterTag {
				return true
			}
		}
	}
	return false
}
