package cmds

import (
	"bufio"
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewPRCommand() *cobra.Command {
	var (
		workspace string
		dryRun    bool
		force     bool
		draft     bool
		title     string
		body      string
	)

	cmd := &cobra.Command{
		Use:   "pr [workspace-name]",
		Short: "Create pull requests for workspace branches",
		Long: `Create pull requests using 'gh pr create' for branches in the workspace that need PRs.

This command will:
1. Check each repository in the workspace for branches that could use PRs
2. Ask for confirmation before creating each PR (unless --force is used)
3. Use 'gh pr create' to create the pull requests

A branch is considered to need a PR if:
- It's not the main/master branch
- It's not merged to origin/main yet
- It has commits ahead of origin/main
- If the branch doesn't exist on remote, it will be pushed first

Requirements:
- GitHub CLI (gh) must be installed and authenticated
- Repositories must be hosted on GitHub

Examples:
  # Check what PRs would be created (dry run)
  workspace-manager pr my-workspace --dry-run

  # Create PRs interactively
  workspace-manager pr my-workspace

  # Create all PRs without asking
  workspace-manager pr my-workspace --force

  # Create draft PRs with custom title
  workspace-manager pr my-workspace --draft --title "WIP: Feature branch"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := workspace
			if len(args) > 0 {
				workspaceName = args[0]
			}
			return runPR(cmd.Context(), workspaceName, dryRun, force, draft, title, body)
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what PRs would be created without actually creating them")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Create PRs without asking for confirmation")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create draft pull requests")
	cmd.Flags().StringVar(&title, "title", "", "Custom title for all PRs (default: use branch name)")
	cmd.Flags().StringVar(&body, "body", "", "Custom body for all PRs")

	return cmd
}

func runPR(ctx context.Context, workspaceName string, dryRun, force, draft bool, customTitle, customBody string) error {
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
			return errors.Wrap(err, "failed to detect workspace. Use 'workspace-manager pr <workspace-name>' or specify --workspace flag")
		}
		workspaceName = detected
	}

	// Load workspace
	workspace, err := loadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	// Get workspace status to check branch merge status
	checker := wsm.NewStatusChecker()
	status, err := checker.GetWorkspaceStatus(ctx, workspace)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace status")
	}

	// Find branches that need PRs
	var candidateBranches []PRCandidate
	for _, repoStatus := range status.Repositories {
		if candidate, needsPR := checkIfNeedsPR(ctx, repoStatus, workspace.Path); needsPR {
			candidateBranches = append(candidateBranches, candidate)
		}
	}

	if len(candidateBranches) == 0 {
		fmt.Println("No branches found that need pull requests.")
		return nil
	}

	// Show what we found
	fmt.Printf("Found %d branch(es) that could use pull requests:\n\n", len(candidateBranches))

	for i, candidate := range candidateBranches {
		fmt.Printf("%d. %s/%s\n", i+1, candidate.Repository, candidate.Branch)
		fmt.Printf("   Commits ahead: %d\n", candidate.CommitsAhead)
		if candidate.NeedsPush {
			fmt.Printf("   🚀 Needs push: Branch must be pushed to remote first\n")
		}
		if candidate.ExistingPR != "" {
			fmt.Printf("   ⚠️  Existing PR: %s\n", candidate.ExistingPR)
		}
		fmt.Printf("   Remote URL: %s\n", candidate.RemoteURL)
		fmt.Println()
	}

	if dryRun {
		fmt.Println("Dry run mode - no PRs will be created.")
		return nil
	}

	// Create PRs
	reader := bufio.NewReader(os.Stdin)
	for _, candidate := range candidateBranches {
		if candidate.ExistingPR != "" {
			fmt.Printf("Skipping %s/%s - PR already exists: %s\n", candidate.Repository, candidate.Branch, candidate.ExistingPR)
			continue
		}

		shouldCreate := force
		if !force {
			fmt.Printf("Create PR for %s/%s? [y/N]: ", candidate.Repository, candidate.Branch)
			response, _ := reader.ReadString('\n')
			response = strings.ToLower(strings.TrimSpace(response))
			shouldCreate = response == "y" || response == "yes"
		}

		if shouldCreate {
			// Push branch first if needed
			if candidate.NeedsPush {
				fmt.Printf("🚀 Pushing branch %s/%s to remote...\n", candidate.Repository, candidate.Branch)
				if err := pushBranchForPR(ctx, candidate); err != nil {
					fmt.Printf("❌ Failed to push branch %s/%s: %v\n", candidate.Repository, candidate.Branch, err)
					continue
				}
				fmt.Printf("✅ Pushed branch %s/%s\n", candidate.Repository, candidate.Branch)
			}

			if err := createPR(ctx, candidate, draft, customTitle, customBody); err != nil {
				fmt.Printf("❌ Failed to create PR for %s/%s: %v\n", candidate.Repository, candidate.Branch, err)
			} else {
				fmt.Printf("✅ Created PR for %s/%s\n", candidate.Repository, candidate.Branch)
			}
		} else {
			fmt.Printf("Skipped %s/%s\n", candidate.Repository, candidate.Branch)
		}
	}

	return nil
}

type PRCandidate struct {
	Repository   string
	Branch       string
	RepoPath     string
	CommitsAhead int
	RemoteURL    string
	ExistingPR   string // URL if PR already exists
	NeedsPush    bool   // true if branch needs to be pushed to remote first
}

func checkGHCLI(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "gh", "--version")
	if err := cmd.Run(); err != nil {
		return errors.New("GitHub CLI (gh) is not installed or not in PATH. Please install it from https://cli.github.com/")
	}

	// Check if authenticated
	cmd = exec.CommandContext(ctx, "gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return errors.New("GitHub CLI is not authenticated. Please run 'gh auth login' first")
	}

	return nil
}

func checkIfNeedsPR(ctx context.Context, repoStatus wsm.RepositoryStatus, workspacePath string) (PRCandidate, bool) {
	candidate := PRCandidate{
		Repository: repoStatus.Repository.Name,
		Branch:     repoStatus.CurrentBranch,
		RepoPath:   filepath.Join(workspacePath, repoStatus.Repository.Name),
		RemoteURL:  repoStatus.Repository.RemoteURL,
	}

	log.Debug().
		Str("repository", candidate.Repository).
		Str("branch", candidate.Branch).
		Str("repoPath", candidate.RepoPath).
		Msg("Checking if repository needs a PR")

	// Skip if no current branch
	if repoStatus.CurrentBranch == "" {
		log.Debug().Str("repository", candidate.Repository).Msg("Skipping: no current branch")
		return candidate, false
	}

	// Skip main/master branches
	if repoStatus.CurrentBranch == "main" || repoStatus.CurrentBranch == "master" {
		log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Msg("Skipping: is main/master branch")
		return candidate, false
	}

	// Skip if already merged to origin/main
	if repoStatus.IsMerged {
		log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Msg("Skipping: already merged to origin/main")
		return candidate, false
	}

	// Get ahead/behind counts against origin/main specifically for PR purposes
	aheadCount, behindCount, err := getAheadBehindOriginMain(ctx, candidate.RepoPath)
	if err != nil {
		log.Debug().Err(err).Str("repository", candidate.Repository).Str("branch", candidate.Branch).Msg("Failed to get ahead/behind counts against origin/main")
		// Fall back to the status ahead count
		aheadCount = repoStatus.Ahead
	}

	candidate.CommitsAhead = aheadCount
	log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Int("ahead", aheadCount).Int("behind", behindCount).Msg("Repository commits against origin/main")

	// Skip if no commits ahead of origin/main
	if aheadCount == 0 {
		log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Msg("Skipping: no commits ahead of origin/main")
		return candidate, false
	}

	// Check if branch exists on remote
	branchExists := branchExistsOnRemote(ctx, candidate.RepoPath, repoStatus.CurrentBranch)
	log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Bool("exists", branchExists).Msg("Checked if branch exists on remote")

	// If branch doesn't exist on remote but has commits ahead, we need to push first
	if !branchExists {
		log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Msg("Branch needs to be pushed before creating PR")
		candidate.NeedsPush = true
	}

	// Check if PR already exists
	if existingPR := checkExistingPR(ctx, candidate.RepoPath, repoStatus.CurrentBranch); existingPR != "" {
		log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Str("existingPR", existingPR).Msg("Found existing PR")
		candidate.ExistingPR = existingPR
	} else {
		log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Msg("No existing PR found")
	}

	log.Debug().Str("repository", candidate.Repository).Str("branch", candidate.Branch).Msg("Repository NEEDS a PR")
	return candidate, true
}

func getAheadBehindOriginMain(ctx context.Context, repoPath string) (int, int, error) {
	// Get ahead/behind counts against origin/main
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--left-right", "--count", "HEAD...origin/main")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Str("repoPath", repoPath).Msg("Failed to get ahead/behind counts against origin/main")
		return 0, 0, err
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0, errors.New("unexpected git rev-list output")
	}

	ahead := 0
	behind := 0

	if aheadVal, err := strconv.Atoi(parts[0]); err == nil {
		ahead = aheadVal
	}

	if behindVal, err := strconv.Atoi(parts[1]); err == nil {
		behind = behindVal
	}

	log.Debug().Str("repoPath", repoPath).Int("ahead", ahead).Int("behind", behind).Msg("Got ahead/behind counts against origin/main")
	return ahead, behind, nil
}

func branchExistsOnRemote(ctx context.Context, repoPath, branch string) bool {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", "origin", branch)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
}

func checkExistingPR(ctx context.Context, repoPath, branch string) string {
	cmd := exec.CommandContext(ctx, "gh", "pr", "list", "--head", branch, "--json", "url", "--jq", ".[0].url")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func pushBranchForPR(ctx context.Context, candidate PRCandidate) error {
	cmd := exec.CommandContext(ctx, "git", "push", "-u", "origin", candidate.Branch)
	cmd.Dir = candidate.RepoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "git push failed: %s", string(output))
	}

	return nil
}

func createPR(ctx context.Context, candidate PRCandidate, draft bool, customTitle, customBody string) error {
	args := []string{"pr", "create"}

	// Add title
	title := customTitle
	if title == "" {
		title = fmt.Sprintf("Feature: %s", candidate.Branch)
	}
	args = append(args, "--title", title)

	// Add body
	body := customBody
	if body == "" {
		body = fmt.Sprintf("Pull request for branch: %s\n\nCreated automatically by workspace-manager.", candidate.Branch)
	}
	args = append(args, "--body", body)

	// Add draft flag if requested
	if draft {
		args = append(args, "--draft")
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = candidate.RepoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "gh pr create failed: %s", string(output))
	}

	return nil
}
