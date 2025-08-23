package wsm

import (
    "context"
    "os"

    "github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
)

// BuildGitBackends constructs a GitClient and WorktreeManager based on env/config.
// Backend precedence (env WSM_GIT_BACKEND): hybrid (default), gogit, cli.
func BuildGitBackends(ctx context.Context) (gitclient.GitClient, gitclient.WorktreeManager) {
    backend := os.Getenv("WSM_GIT_BACKEND")
    if backend == "" { backend = "hybrid" }

    switch backend {
    case "gogit":
        return gitclient.NewGoGit(), gitclient.NewCliWorktrees()
    case "cli":
        return gitclient.NewCli(), gitclient.NewCliWorktrees()
    default: // hybrid
        return gitclient.NewHybrid(gitclient.NewGoGit(), gitclient.NewCli()), gitclient.NewCliWorktrees()
    }
}


