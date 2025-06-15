package cmds

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewPushCommand() *cobra.Command {
	var (
		workspace   string
		dryRun      bool
		force       bool
		setUpstream bool
	)

	cmd := &cobra.Command{
		Use:   "push <remote-name> [workspace-name]",
		Short: "Push workspace branches to specified remote",
		Long: `Push branches in the workspace to a specified remote (typically a fork).

This command will:
1. Check each repository in the workspace for branches that need to be pushed
2. Use 'gh repo view' to verify the remote repository exists  
3. Ask for confirmation before pushing each branch (unless --force is used)
4. Push branches to the specified remote

A branch is considered to need pushing if:
- It has local commits that aren't on the remote yet
- It's not the main/master branch (unless it has unpushed commits)
- The repository exists on GitHub

Requirements:
- GitHub CLI (gh) must be installed and authenticated
- Repositories must be hosted on GitHub
- The specified remote must exist and be accessible

Examples:
  # Check what would be pushed (dry run)  
  workspace-manager push fork my-workspace --dry-run

  # Push to fork remote interactively
  workspace-manager push fork my-workspace

  # Push all branches without asking
  workspace-manager push fork my-workspace --force

  # Push and set upstream tracking
  workspace-manager push fork my-workspace --set-upstream`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			remoteName := args[0]
			workspaceName := workspace
			if len(args) > 1 {
				workspaceName = args[1]
			}
			return runPush(cmd.Context(), remoteName, workspaceName, dryRun, force, setUpstream)
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be pushed without actually pushing")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Push without asking for confirmation")
	cmd.Flags().BoolVarP(&setUpstream, "set-upstream", "u", false, "Set upstream tracking for pushed branches")

	return cmd
}

func runPush(ctx context.Context, remoteName, workspaceName string, dryRun, force, setUpstream bool) error {
	// Check if gh CLI is available
	if err := checkGHCLI(ctx); err != nil {
		return err
	}

	// If no workspace specified, try to detect current workspace
	if workspaceName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to get current directory")
		}

		detected, err := detectWorkspace(cwd)
		if err != nil {
			return errors.Wrap(err, "failed to detect workspace. Use 'workspace-manager push <remote> <workspace-name>' or specify --workspace flag")
		}
		workspaceName = detected
	}

	// Load workspace
	workspace, err := loadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	// Get workspace status
	checker := wsm.NewStatusChecker()
	status, err := checker.GetWorkspaceStatus(ctx, workspace)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace status")
	}

	// Find branches that need pushing
	var candidateBranches []PushCandidate
	for _, repoStatus := range status.Repositories {
		if candidate, needsPush := checkIfNeedsPush(ctx, repoStatus, workspace.Path, remoteName); needsPush {
			candidateBranches = append(candidateBranches, candidate)
		}
	}

	if len(candidateBranches) == 0 {
		fmt.Printf("No branches found that need pushing to remote '%s'.\n", remoteName)
		return nil
	}

	// Show what we found
	fmt.Printf("Found %d branch(es) that could be pushed to remote '%s':\n\n", len(candidateBranches), remoteName)

	for i, candidate := range candidateBranches {
		fmt.Printf("%d. %s/%s\n", i+1, candidate.Repository, candidate.Branch)
		fmt.Printf("   Local commits: %d\n", candidate.LocalCommits)
		fmt.Printf("   Target remote: %s/%s\n", remoteName, candidate.RemoteRepo)
		if candidate.RemoteExists {
			fmt.Printf("   Remote branch exists: %t\n", candidate.RemoteBranchExists)
		} else {
			fmt.Printf("   ⚠️  Remote repository not found or not accessible\n")
		}
		fmt.Println()
	}

	if dryRun {
		fmt.Println("Dry run mode - no branches will be pushed.")
		return nil
	}

	// Push branches
	reader := bufio.NewReader(os.Stdin)
	for _, candidate := range candidateBranches {
		if !candidate.RemoteExists {
			fmt.Printf("Skipping %s/%s - remote repository '%s' not found or not accessible\n",
				candidate.Repository, candidate.Branch, candidate.RemoteRepo)
			continue
		}

		shouldPush := force
		if !force {
			fmt.Printf("Push %s/%s to %s? [y/N]: ", candidate.Repository, candidate.Branch, remoteName)
			response, _ := reader.ReadString('\n')
			response = strings.ToLower(strings.TrimSpace(response))
			shouldPush = response == "y" || response == "yes"
		}

		if shouldPush {
			if err := pushBranch(ctx, candidate, remoteName, setUpstream); err != nil {
				fmt.Printf("❌ Failed to push %s/%s: %v\n", candidate.Repository, candidate.Branch, err)
			} else {
				fmt.Printf("✅ Pushed %s/%s to %s\n", candidate.Repository, candidate.Branch, remoteName)
			}
		} else {
			fmt.Printf("Skipped %s/%s\n", candidate.Repository, candidate.Branch)
		}
	}

	return nil
}

type PushCandidate struct {
	Repository         string
	Branch             string
	RepoPath           string
	LocalCommits       int
	RemoteRepo         string // The remote repository name (owner/repo)
	RemoteExists       bool   // Whether the remote repository exists
	RemoteBranchExists bool   // Whether the branch exists on the remote
}

type RepoInfo struct {
	NameWithOwner    string `json:"nameWithOwner"`
	URL              string `json:"url"`
	DefaultBranchRef struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
}

func checkIfNeedsPush(ctx context.Context, repoStatus wsm.RepositoryStatus, workspacePath, remoteName string) (PushCandidate, bool) {
	candidate := PushCandidate{
		Repository: repoStatus.Repository.Name,
		Branch:     repoStatus.CurrentBranch,
		RepoPath:   filepath.Join(workspacePath, repoStatus.Repository.Name),
	}

	log.Debug().
		Str("repository", candidate.Repository).
		Str("branch", candidate.Branch).
		Str("remote", remoteName).
		Msg("Checking if repository branch needs pushing")

	// Skip if no current branch
	if repoStatus.CurrentBranch == "" {
		log.Debug().Str("repository", candidate.Repository).Msg("Skipping: no current branch")
		return candidate, false
	}

	// Get repository info from GitHub
	repoInfo, err := getRepoInfo(ctx, candidate.RepoPath)
	if err != nil {
		log.Debug().Err(err).Str("repository", candidate.Repository).Msg("Failed to get repository info")
		return candidate, false
	}

	candidate.RemoteRepo = repoInfo.NameWithOwner
	candidate.RemoteExists = true

	// Check if remote repository exists (by trying to access it)
	if !checkRemoteRepoExists(ctx, remoteName, repoInfo.NameWithOwner) {
		log.Debug().Str("repository", candidate.Repository).Str("remote", remoteName).Str("remoteRepo", repoInfo.NameWithOwner).Msg("Remote repository not accessible")
		candidate.RemoteExists = false
		// Still return as candidate so user can see the issue
	}

	// Get local commits that aren't pushed to the remote yet
	localCommits, err := getLocalCommits(ctx, candidate.RepoPath, remoteName, candidate.Branch)
	if err != nil {
		log.Debug().Err(err).Str("repository", candidate.Repository).Str("branch", candidate.Branch).Msg("Failed to get local commits")
		// If we can't determine local commits, assume there might be some
		localCommits = 1
	}

	candidate.LocalCommits = localCommits
	log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Int("localCommits", localCommits).Msg("Found local commits")

	// Check if remote branch exists
	if candidate.RemoteExists {
		candidate.RemoteBranchExists = checkRemoteBranchExists(ctx, candidate.RepoPath, remoteName, candidate.Branch)
		log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Bool("remoteBranchExists", candidate.RemoteBranchExists).Msg("Checked remote branch existence")
	}

	// Need to push if we have local commits
	needsPush := localCommits > 0

	if needsPush {
		log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Msg("Branch NEEDS pushing")
	} else {
		log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Msg("Branch does not need pushing")
	}

	return candidate, needsPush
}

func getRepoInfo(ctx context.Context, repoPath string) (*RepoInfo, error) {
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner,url,defaultBranchRef")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get repository info from GitHub")
	}

	var info RepoInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, errors.Wrap(err, "failed to parse repository info")
	}

	log.Debug().Str("repoPath", repoPath).Str("nameWithOwner", info.NameWithOwner).Msg("Got repository info")
	return &info, nil
}

func checkRemoteRepoExists(ctx context.Context, remoteName, repoFullName string) bool {
	// The repoFullName is already in "owner/repo" format, so we need to replace the owner with remoteName
	parts := strings.Split(repoFullName, "/")
	if len(parts) != 2 {
		log.Debug().Str("repoFullName", repoFullName).Msg("Invalid repository name format")
		return false
	}

	// Construct remote repo as remoteName/repoName
	remoteRepo := fmt.Sprintf("%s/%s", remoteName, parts[1])

	// Try to access the remote repository
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", remoteRepo)
	err := cmd.Run()

	log.Debug().Str("remoteName", remoteName).Str("repoFullName", repoFullName).Str("remoteRepo", remoteRepo).Bool("exists", err == nil).Msg("Checked remote repository existence")
	return err == nil
}

func getLocalCommits(ctx context.Context, repoPath, remoteName, branch string) (int, error) {
	// Check if remote branch exists first
	remoteRef := fmt.Sprintf("%s/%s", remoteName, branch)

	// Try to get commits ahead of remote branch (local commits that aren't on remote)
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("%s..HEAD", remoteRef))
	cmd.Dir = repoPath
	output, err := cmd.Output()

	if err != nil {
		// Remote branch might not exist, check if we have any commits to push
		// by comparing against origin/main or just counting local commits
		log.Debug().Err(err).Str("repoPath", repoPath).Str("remoteRef", remoteRef).Msg("Remote branch not found, checking against origin/main")

		// Try to compare against origin/main
		cmd = exec.CommandContext(ctx, "git", "rev-list", "--count", "origin/main..HEAD")
		cmd.Dir = repoPath
		output, err = cmd.Output()
		if err != nil {
			// Fallback: count commits on current branch
			cmd = exec.CommandContext(ctx, "git", "rev-list", "--count", "HEAD")
			cmd.Dir = repoPath
			output, err = cmd.Output()
			if err != nil {
				return 0, err
			}
		}
	}

	count := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count); err != nil {
		return 0, err
	}

	log.Debug().Str("repoPath", repoPath).Str("remoteName", remoteName).Str("branch", branch).Int("localCommits", count).Msg("Got local commits count")
	return count, nil
}

func checkRemoteBranchExists(ctx context.Context, repoPath, remoteName, branch string) bool {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", remoteName, branch)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

func pushBranch(ctx context.Context, candidate PushCandidate, remoteName string, setUpstream bool) error {
	args := []string{"push"}

	if setUpstream {
		args = append(args, "-u")
	}

	args = append(args, remoteName, candidate.Branch)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = candidate.RepoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "git push failed: %s", string(output))
	}

	log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Str("remote", remoteName).Msg("Successfully pushed branch")
	return nil
}
