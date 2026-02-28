package workflows

import (
	"sort"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
)

// ListWorkflow orchestrates repository/workspace listing.
type ListWorkflow struct {
	manager *wsm.WorkspaceManager
}

// NewListWorkflow creates a list workflow.
func NewListWorkflow() (*ListWorkflow, error) {
	manager, err := wsm.NewWorkspaceManager()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace manager")
	}
	return &ListWorkflow{manager: manager}, nil
}

// ListRepositories returns repositories optionally filtered by tags.
func (lw *ListWorkflow) ListRepositories(tags []string) ([]wsm.Repository, error) {
	return lw.manager.Discoverer.GetRepositoriesByTags(tags), nil
}

// ListWorkspaces returns all workspaces sorted newest first.
func (lw *ListWorkflow) ListWorkspaces() ([]wsm.Workspace, error) {
	workspaces, err := wsm.LoadWorkspaces()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load workspaces")
	}

	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].Created.After(workspaces[j].Created)
	})

	return workspaces, nil
}
