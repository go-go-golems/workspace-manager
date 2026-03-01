package wsm

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

// HistoryOperations handles workspace history/log retrieval.
type HistoryOperations struct {
	workspace *Workspace
}

// NewHistoryOperations creates a history operations service.
func NewHistoryOperations(workspace *Workspace) *HistoryOperations {
	return &HistoryOperations{workspace: workspace}
}

// GetWorkspaceLog returns log output for each repository that has output.
func (ho *HistoryOperations) GetWorkspaceLog(ctx context.Context, since string, oneline bool, limit int) (map[string]string, error) {
	logs := make(map[string]string)

	for _, repo := range ho.workspace.Repositories {
		repoPath := filepath.Join(ho.workspace.Path, repo.Name)
		repoLog, err := ho.getRepositoryLog(ctx, repoPath, since, oneline, limit)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get log for %s", repo.Name)
		}
		if repoLog != "" {
			logs[repo.Name] = repoLog
		}
	}

	return logs, nil
}

func (ho *HistoryOperations) getRepositoryLog(ctx context.Context, repoPath, since string, oneline bool, limit int) (string, error) {
	args := []string{"log"}

	if since != "" {
		args = append(args, "--since", since)
	}
	if oneline {
		args = append(args, "--oneline")
	}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(out), nil
}
