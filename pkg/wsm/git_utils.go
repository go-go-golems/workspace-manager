package wsm

import (
	"context"
	"os/exec"
	"strings"

	branchsvc "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"github.com/rs/zerolog/log"
)

// getGitCurrentBranch returns the current branch name
func getGitCurrentBranch(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// CheckBranchMerged checks if the current branch has been merged to the configured remote base branch.
func CheckBranchMerged(ctx context.Context, path string, baseBranch string) (bool, error) {
	base := branchsvc.ResolveBaseBranch(baseBranch)
	remoteBase := branchsvc.RemoteTrackingRef(branchsvc.DefaultRemoteName, base)

	// Get current branch for logging
	currentBranch, branchErr := getGitCurrentBranch(ctx, path)
	if branchErr != nil {
		log.Debug().Err(branchErr).Str("path", path).Msg("Failed to get current branch for merge check")
		currentBranch = "unknown"
	}

	log.Debug().
		Str("path", path).
		Str("branch", currentBranch).
		Str("upstream", remoteBase).
		Str("base_branch", string(base)).
		Msg("Checking if branch is merged to configured remote base")

	// Check if HEAD has been merged into the configured default remote base branch.
	// This command returns 0 if the current HEAD is merged, non-zero otherwise
	cmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", "HEAD", remoteBase)
	cmd.Dir = path
	err := cmd.Run()

	merged := err == nil
	log.Debug().Str("path", path).Str("branch", currentBranch).Bool("merged", merged).Msg("Branch merge check result")

	return merged, nil
}

// CheckBranchNeedsRebase checks if the current branch needs to be rebased on the configured remote base branch.
func CheckBranchNeedsRebase(ctx context.Context, path string, baseBranch string) (bool, error) {
	base := branchsvc.ResolveBaseBranch(baseBranch)
	remoteBase := branchsvc.RemoteTrackingRef(branchsvc.DefaultRemoteName, base)

	// Get current branch for logging
	currentBranch, branchErr := getGitCurrentBranch(ctx, path)
	if branchErr != nil {
		log.Debug().Err(branchErr).Str("path", path).Msg("Failed to get current branch for rebase check")
		currentBranch = "unknown"
	}

	// Skip rebase check if we're already on the configured base branch.
	if currentBranch == string(base) || (string(base) == "main" && currentBranch == "master") {
		log.Debug().
			Str("path", path).
			Str("branch", currentBranch).
			Str("base_branch", string(base)).
			Msg("Skipping rebase check - already on base branch")
		return false, nil
	}

	log.Debug().
		Str("path", path).
		Str("branch", currentBranch).
		Str("upstream", remoteBase).
		Str("base_branch", string(base)).
		Msg("Checking if branch needs rebase on configured remote base")

	// Check if remote base branch has new commits compared to the merge-base.
	// This tells us if the base branch moved forward since we branched.
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", "HEAD.."+remoteBase)
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("Failed to check for commits ahead on default remote main")
		return false, err
	}

	commitCount := strings.TrimSpace(string(output))
	needsRebase := commitCount != "0"
	log.Debug().Str("path", path).Str("branch", currentBranch).Str("commits_behind", commitCount).Bool("needs_rebase", needsRebase).Msg("Branch rebase check result")

	return needsRebase, nil
}
