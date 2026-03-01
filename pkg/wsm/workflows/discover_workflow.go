package workflows

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
)

// DiscoverRequest captures repository discovery options.
type DiscoverRequest struct {
	Paths     []string
	Recursive bool
	MaxDepth  int
}

// DiscoverResult captures normalized discovery input and resulting repo count.
type DiscoverResult struct {
	Paths           []string
	RepositoryCount int
}

// DiscoverWorkflow orchestrates repository discovery.
type DiscoverWorkflow struct {
	manager *wsm.WorkspaceManager
}

// NewDiscoverWorkflow creates a discovery workflow.
func NewDiscoverWorkflow() (*DiscoverWorkflow, error) {
	manager, err := wsm.NewWorkspaceManager()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace manager")
	}
	return &DiscoverWorkflow{manager: manager}, nil
}

// Discover resolves paths, runs discovery, and returns the resulting repository count.
func (dw *DiscoverWorkflow) Discover(ctx context.Context, req DiscoverRequest) (*DiscoverResult, error) {
	expandedPaths, err := ResolveDiscoverPaths(req.Paths)
	if err != nil {
		return nil, err
	}

	if err := dw.manager.Discoverer.DiscoverRepositories(ctx, expandedPaths, req.Recursive, req.MaxDepth); err != nil {
		return nil, errors.Wrap(err, "discovery failed")
	}

	repos := dw.manager.Discoverer.GetRepositories()
	return &DiscoverResult{Paths: expandedPaths, RepositoryCount: len(repos)}, nil
}

// ResolveDiscoverPaths expands and validates discovery paths.
func ResolveDiscoverPaths(paths []string) ([]string, error) {
	if len(paths) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current directory")
		}
		paths = []string{cwd}
	}

	expandedPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		expandedPath, err := expandPath(path)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			return nil, errors.Errorf("path does not exist: %s", expandedPath)
		}
		expandedPaths = append(expandedPaths, expandedPath)
	}

	return expandedPaths, nil
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.Wrap(err, "failed to get home directory")
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get absolute path for %s", path)
	}

	return absPath, nil
}
