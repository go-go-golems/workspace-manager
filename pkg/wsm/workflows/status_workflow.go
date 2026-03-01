package workflows

import (
	"context"
	"os"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
)

// StatusRequest captures status lookup options.
type StatusRequest struct {
	WorkspaceName string
	Jobs          int
	Fetch         bool
}

// StatusWorkflow resolves workspace status requests.
type StatusWorkflow struct {
	workspaceContext *wsm.WorkspaceContextService
	checker          *wsm.StatusChecker
}

// NewStatusWorkflow creates a status workflow service.
func NewStatusWorkflow() *StatusWorkflow {
	return &StatusWorkflow{
		workspaceContext: wsm.NewWorkspaceContextService(),
		checker:          wsm.NewStatusChecker(),
	}
}

// GetStatus resolves workspace context and fetches status.
func (sw *StatusWorkflow) GetStatus(ctx context.Context, req StatusRequest) (*wsm.WorkspaceStatus, error) {
	workspaceName := req.WorkspaceName
	if workspaceName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current directory")
		}
		detected, err := sw.workspaceContext.DetectWorkspaceName(cwd)
		if err != nil {
			return nil, errors.Wrap(err, "failed to detect workspace. Use 'workspace-manager status <workspace-name>' or specify --workspace flag")
		}
		workspaceName = detected
	}

	workspace, err := sw.workspaceContext.LoadWorkspace(workspaceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	status, err := sw.checker.GetWorkspaceStatusWithOptions(ctx, workspace, wsm.StatusOptions{
		MaxJobs: req.Jobs,
		Fetch:   req.Fetch,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace status")
	}

	return status, nil
}
