package wsm

import (
	"context"
	"os/exec"
	"strings"

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

// CheckBranchMerged checks if the current branch has been merged to origin/main
func CheckBranchMerged(ctx context.Context, path string) (bool, error) {
	// Get current branch for logging
	currentBranch, branchErr := getGitCurrentBranch(ctx, path)
	if branchErr != nil {
		log.Debug().Err(branchErr).Str("path", path).Msg("Failed to get current branch for merge check")
		currentBranch = "unknown"
	}

	log.Debug().Str("path", path).Str("branch", currentBranch).Msg("Checking if branch is merged to origin/main")

	// First, fetch to ensure we have latest remote refs
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin", "main")
	fetchCmd.Dir = path
	fetchErr := fetchCmd.Run()
	if fetchErr != nil {
		log.Debug().Err(fetchErr).Str("path", path).Msg("Failed to fetch origin/main - might be offline")
	} else {
		log.Debug().Str("path", path).Msg("Successfully fetched origin/main")
	}

	// Check if HEAD has been merged into origin/main
	// This command returns 0 if the current HEAD is merged, non-zero otherwise
	cmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", "HEAD", "origin/main")
	cmd.Dir = path
	err := cmd.Run()

	merged := err == nil
	log.Debug().Str("path", path).Str("branch", currentBranch).Bool("merged", merged).Msg("Branch merge check result")

	return merged, nil
}

// CheckBranchNeedsRebase checks if the current branch needs to be rebased on origin/main
func CheckBranchNeedsRebase(ctx context.Context, path string) (bool, error) {
	// Get current branch for logging
	currentBranch, branchErr := getGitCurrentBranch(ctx, path)
	if branchErr != nil {
		log.Debug().Err(branchErr).Str("path", path).Msg("Failed to get current branch for rebase check")
		currentBranch = "unknown"
	}

	// Skip rebase check if we're on main branch
	if currentBranch == "main" || currentBranch == "master" {
		log.Debug().Str("path", path).Str("branch", currentBranch).Msg("Skipping rebase check - already on main branch")
		return false, nil
	}

	log.Debug().Str("path", path).Str("branch", currentBranch).Msg("Checking if branch needs rebase on origin/main")

	// First, fetch to ensure we have latest remote refs
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin", "main")
	fetchCmd.Dir = path
	fetchErr := fetchCmd.Run()
	if fetchErr != nil {
		log.Debug().Err(fetchErr).Str("path", path).Msg("Failed to fetch origin/main - might be offline")
	} else {
		log.Debug().Str("path", path).Msg("Successfully fetched origin/main")
	}

	// Check if origin/main has new commits compared to the merge-base
	// This tells us if origin/main has moved forward since we branched
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", "HEAD..origin/main")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("Failed to check for commits ahead on origin/main")
		return false, err
	}

	commitCount := strings.TrimSpace(string(output))
	needsRebase := commitCount != "0"
	log.Debug().Str("path", path).Str("branch", currentBranch).Str("commits_behind", commitCount).Bool("needs_rebase", needsRebase).Msg("Branch rebase check result")

	return needsRebase, nil
}
