package wsm

import (
	"context"

	branchsvc "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
)

// BuildGitBackends constructs the CLI-backed GitClient and WorktreeManager.
func BuildGitBackends(ctx context.Context) (gitclient.GitClient, gitclient.WorktreeManager) {
	return gitclient.NewCli(), gitclient.NewCliWorktrees()
}

// BuildBranchService constructs the branch policy service from the active git backend.
func BuildBranchService(ctx context.Context) branchsvc.Service {
	gc, _ := BuildGitBackends(ctx)
	return branchsvc.NewService(gc, branchsvc.DefaultRemoteName)
}
