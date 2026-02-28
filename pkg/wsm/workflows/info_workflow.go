package workflows

import (
	"os"
	"strconv"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
)

// InfoWorkflow resolves workspace info requests.
type InfoWorkflow struct {
	workspaceContext *wsm.WorkspaceContextService
}

// NewInfoWorkflow creates an info workflow service.
func NewInfoWorkflow() *InfoWorkflow {
	return &InfoWorkflow{workspaceContext: wsm.NewWorkspaceContextService()}
}

// ResolveWorkspace resolves the effective workspace (explicit or from current directory).
func (iw *InfoWorkflow) ResolveWorkspace(workspaceName string) (*wsm.Workspace, error) {
	resolvedName := workspaceName
	if resolvedName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current directory")
		}

		detected, err := iw.workspaceContext.DetectWorkspaceName(cwd)
		if err != nil {
			return nil, errors.Wrap(err, "failed to detect workspace. Use 'workspace-manager info <workspace-name>' or specify --workspace flag")
		}
		resolvedName = detected
	}

	workspace, err := iw.workspaceContext.LoadWorkspace(resolvedName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load workspace '%s'", resolvedName)
	}
	return workspace, nil
}

// FieldValue returns a string value for a supported workspace field.
func (iw *InfoWorkflow) FieldValue(workspace *wsm.Workspace, field string) (string, error) {
	switch strings.ToLower(field) {
	case "path":
		return workspace.Path, nil
	case "name":
		return workspace.Name, nil
	case "branch":
		return workspace.Branch, nil
	case "repositories":
		return strconv.Itoa(len(workspace.Repositories)), nil
	case "created":
		return workspace.Created.Format("2006-01-02 15:04:05"), nil
	case "date":
		return workspace.Created.Format("2006-01-02"), nil
	case "time":
		return workspace.Created.Format("15:04:05"), nil
	default:
		return "", errors.Errorf("unknown field: %s. Available fields: path, name, branch, repositories, created, date, time", field)
	}
}
