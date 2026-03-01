package wsm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// WorkspaceContextService centralizes workspace detection and loading behavior.
// Commands should use this service instead of implementing their own heuristics.
type WorkspaceContextService struct{}

// NewWorkspaceContextService creates a workspace context service.
func NewWorkspaceContextService() *WorkspaceContextService {
	return &WorkspaceContextService{}
}

// LoadWorkspace loads a workspace by name from persisted workspace configurations.
func (s *WorkspaceContextService) LoadWorkspace(name string) (*Workspace, error) {
	workspaces, err := LoadWorkspaces()
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

// DetectCurrentWorkspace returns the workspace containing cwd.
// This method intentionally does not guess unknown names; it only returns persisted workspaces.
func (s *WorkspaceContextService) DetectCurrentWorkspace(cwd string) (*Workspace, error) {
	workspaces, err := LoadWorkspaces()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load workspaces")
	}

	for _, workspace := range workspaces {
		if strings.HasPrefix(cwd, workspace.Path) {
			return &workspace, nil
		}
		for _, repo := range workspace.Repositories {
			repoWorktreePath := filepath.Join(workspace.Path, repo.Name)
			if strings.HasPrefix(cwd, repoWorktreePath) {
				return &workspace, nil
			}
		}
	}

	return nil, errors.New("not in a workspace directory")
}

// DetectWorkspaceName detects a workspace name from cwd.
// It first checks persisted workspace paths, then falls back to structural heuristics.
func (s *WorkspaceContextService) DetectWorkspaceName(cwd string) (string, error) {
	log.Debug().Str("cwd", cwd).Msg("Starting workspace detection")

	workspaces, err := LoadWorkspaces()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to load workspaces")
		return "", errors.Wrap(err, "failed to load workspaces")
	}

	log.Debug().Int("workspaceCount", len(workspaces)).Msg("Loaded workspaces")

	for _, workspace := range workspaces {
		log.Debug().
			Str("workspaceName", workspace.Name).
			Str("workspacePath", workspace.Path).
			Msg("Checking workspace")

		if strings.HasPrefix(cwd, workspace.Path) {
			output.LogInfo(
				fmt.Sprintf("Detected workspace: %s", workspace.Name),
				"Found workspace containing current directory",
				"workspaceName", workspace.Name,
				"workspacePath", workspace.Path,
				"cwd", cwd,
			)
			return workspace.Name, nil
		}

		for _, repo := range workspace.Repositories {
			repoWorktreePath := filepath.Join(workspace.Path, repo.Name)
			log.Debug().
				Str("repo", repo.Name).
				Str("repoWorktreePath", repoWorktreePath).
				Msg("Checking repository worktree path")

			if strings.HasPrefix(cwd, repoWorktreePath) {
				output.LogInfo(
					fmt.Sprintf("Detected workspace: %s (via repo %s)", workspace.Name, repo.Name),
					"Found workspace via repository worktree path",
					"workspaceName", workspace.Name,
					"repo", repo.Name,
					"repoWorktreePath", repoWorktreePath,
					"cwd", cwd,
				)
				return workspace.Name, nil
			}
		}
	}

	log.Debug().Msg("No workspace found containing current directory, trying heuristic detection")

	dir := cwd
	for {
		log.Debug().Str("dir", dir).Msg("Checking directory for workspace structure")

		entries, err := os.ReadDir(dir)
		if err != nil {
			log.Debug().Err(err).Str("dir", dir).Msg("Failed to read directory")
			return "", err
		}

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

		if gitDirs >= 2 {
			dirName := filepath.Base(dir)
			log.Debug().Str("dirName", dirName).Msg("Checking if directory name matches any workspace")

			for _, workspace := range workspaces {
				if workspace.Name == dirName || strings.Contains(workspace.Path, dirName) {
					output.LogInfo(
						fmt.Sprintf("Detected workspace: %s", workspace.Name),
						"Found workspace by directory name match",
						"workspaceName", workspace.Name,
						"dirName", dirName,
					)
					return workspace.Name, nil
				}
			}

			log.Debug().Str("dirName", dirName).Msg("Using directory name as workspace name")
			return dirName, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			log.Debug().Msg("Reached filesystem root")
			break
		}
		dir = parent
	}

	log.Debug().Msg("No workspace detected")
	return "", errors.New("not in a workspace directory")
}
