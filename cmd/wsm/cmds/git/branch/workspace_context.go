package branch

import (
	"os"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
)

func detectCurrentWorkspace() (*wsm.Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current directory")
	}
	ctx := wsm.NewWorkspaceContextService()
	return ctx.DetectCurrentWorkspace(cwd)
}
