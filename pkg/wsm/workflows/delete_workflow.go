package workflows

import (
	"context"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
)

// DeletePreview holds workspace and pre-delete status context.
type DeletePreview struct {
	Workspace   *wsm.Workspace
	Status      *wsm.WorkspaceStatus
	StatusError error
}

// DeleteWorkflow orchestrates workspace deletion flow.
type DeleteWorkflow struct {
	manager *wsm.WorkspaceManager
	checker *wsm.StatusChecker
}

// NewDeleteWorkflow creates a delete workflow.
func NewDeleteWorkflow() (*DeleteWorkflow, error) {
	manager, err := wsm.NewWorkspaceManager()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace manager")
	}

	return &DeleteWorkflow{
		manager: manager,
		checker: wsm.NewStatusChecker(),
	}, nil
}

// Preview loads workspace and status for pre-delete review.
func (dw *DeleteWorkflow) Preview(ctx context.Context, workspaceName string) (*DeletePreview, error) {
	workspace, err := dw.manager.LoadWorkspace(workspaceName)
	if err != nil {
		return nil, errors.Wrapf(err, "workspace '%s' not found", workspaceName)
	}

	status, statusErr := dw.checker.GetWorkspaceStatus(ctx, workspace)
	return &DeletePreview{
		Workspace:   workspace,
		Status:      status,
		StatusError: statusErr,
	}, nil
}

// Delete performs workspace deletion.
func (dw *DeleteWorkflow) Delete(ctx context.Context, workspaceName string, removeFiles bool, forceWorktrees bool) error {
	if err := dw.manager.DeleteWorkspace(ctx, workspaceName, removeFiles, forceWorktrees); err != nil {
		return errors.Wrap(err, "failed to delete workspace")
	}
	return nil
}
