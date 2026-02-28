package cmds

import (
	"os"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
)

func detectWorkspace(cwd string) (string, error) {
	ctx := wsm.NewWorkspaceContextService()
	return ctx.DetectWorkspaceName(cwd)
}

func loadWorkspace(name string) (*wsm.Workspace, error) {
	ctx := wsm.NewWorkspaceContextService()
	return ctx.LoadWorkspace(name)
}

func detectCurrentWorkspace() (*wsm.Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current directory")
	}
	ctx := wsm.NewWorkspaceContextService()
	return ctx.DetectCurrentWorkspace(cwd)
}
