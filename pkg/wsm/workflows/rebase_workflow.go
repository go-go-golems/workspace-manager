package workflows

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	branchsvc "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// RebaseWorkflow handles workspace rebase orchestration.
type RebaseWorkflow struct {
	workspace *wsm.Workspace
}

// NewRebaseWorkflow creates a rebase workflow service.
func NewRebaseWorkflow(workspace *wsm.Workspace) *RebaseWorkflow {
	return &RebaseWorkflow{workspace: workspace}
}

// RebaseRequest describes a rebase run request.
type RebaseRequest struct {
	Repository   string
	TargetBranch string
	Interactive  bool
	DryRun       bool
	Jobs         int
}

// RebaseResult is the result of rebasing one repository.
type RebaseResult struct {
	Repository    string `json:"repository"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
	Rebased       bool   `json:"rebased"`
	Conflicts     bool   `json:"conflicts"`
	CommitsBefore int    `json:"commits_before"`
	CommitsAfter  int    `json:"commits_after"`
	TargetBranch  string `json:"target_branch"`
}

// RebaseStatusRow is one repository row in rebase status.
type RebaseStatusRow struct {
	Repository string
	State      wsm.RebaseState
	Conflicts  int
	Error      string
}

// RebaseActionRow is one repository row for continue/abort actions.
type RebaseActionRow struct {
	Repository string
	Success    bool
	Error      string
}

// ManualPlan returns shell commands to run rebase manually.
func (rw *RebaseWorkflow) ManualPlan(repository, targetBranch string) []string {
	commands := make([]string, 0)
	if repository != "" {
		repoPath := filepath.Join(rw.workspace.Path, repository)
		commands = append(commands, fmt.Sprintf("\n# %s\n(cd %s && git fetch --all && git rebase %s)", repository, repoPath, targetBranch))
		return commands
	}
	for _, repo := range rw.workspace.Repositories {
		repoPath := filepath.Join(rw.workspace.Path, repo.Name)
		commands = append(commands, fmt.Sprintf("\n# %s\n(cd %s && git fetch --all && git rebase %s)", repo.Name, repoPath, targetBranch))
	}
	return commands
}

// Rebase executes rebases for requested repositories.
func (rw *RebaseWorkflow) Rebase(ctx context.Context, req RebaseRequest) ([]RebaseResult, error) {
	jobs := req.Jobs
	if jobs < 1 {
		jobs = 1
	}

	if req.Repository != "" {
		result := rw.rebaseRepository(ctx, req.Repository, req.TargetBranch, req.Interactive, req.DryRun)
		return []RebaseResult{result}, nil
	}

	repos := rw.workspace.Repositories
	if jobs == 1 || len(repos) <= 1 {
		results := make([]RebaseResult, 0, len(repos))
		for _, repo := range repos {
			results = append(results, rw.rebaseRepository(ctx, repo.Name, req.TargetBranch, req.Interactive, req.DryRun))
		}
		return results, nil
	}

	results := make([]RebaseResult, len(repos))
	sem := semaphore.NewWeighted(int64(jobs))
	g, gctx := errgroup.WithContext(ctx)
	for i := range repos {
		i := i
		if err := sem.Acquire(gctx, 1); err != nil {
			return nil, err
		}
		g.Go(func() error {
			defer sem.Release(1)
			repo := repos[i]
			results[i] = rw.rebaseRepository(gctx, repo.Name, req.TargetBranch, req.Interactive, req.DryRun)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// Status returns status rows for rebase state across repositories.
func (rw *RebaseWorkflow) Status(ctx context.Context, repository string, jobs int) ([]RebaseStatusRow, error) {
	repos := rw.filterRepos(repository)
	if jobs < 1 {
		jobs = 1
	}

	rows := make([]RebaseStatusRow, len(repos))
	do := func(i int, innerCtx context.Context) {
		r := repos[i]
		state, conflicts, err := wsm.Status(innerCtx, filepath.Join(rw.workspace.Path, r.Name))
		row := RebaseStatusRow{
			Repository: r.Name,
			State:      state,
			Conflicts:  len(conflicts),
		}
		if err != nil {
			row.Error = err.Error()
		}
		rows[i] = row
	}

	if jobs == 1 || len(repos) <= 1 {
		for i := range repos {
			do(i, ctx)
		}
		return rows, nil
	}

	sem := semaphore.NewWeighted(int64(jobs))
	g, gctx := errgroup.WithContext(ctx)
	for i := range repos {
		i := i
		if err := sem.Acquire(gctx, 1); err != nil {
			return nil, err
		}
		g.Go(func() error {
			defer sem.Release(1)
			do(i, gctx)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return rows, nil
}

// Continue runs rebase --continue across repositories.
func (rw *RebaseWorkflow) Continue(ctx context.Context, repository string, jobs int) ([]RebaseActionRow, error) {
	return rw.runAction(ctx, repository, jobs, wsm.Continue)
}

// Abort runs rebase --abort across repositories.
func (rw *RebaseWorkflow) Abort(ctx context.Context, repository string, jobs int) ([]RebaseActionRow, error) {
	return rw.runAction(ctx, repository, jobs, wsm.Abort)
}

func (rw *RebaseWorkflow) runAction(ctx context.Context, repository string, jobs int, action func(context.Context, string) error) ([]RebaseActionRow, error) {
	repos := rw.filterRepos(repository)
	if jobs < 1 {
		jobs = 1
	}

	rows := make([]RebaseActionRow, len(repos))
	do := func(i int, innerCtx context.Context) {
		r := repos[i]
		repoPath := filepath.Join(rw.workspace.Path, r.Name)
		err := action(innerCtx, repoPath)
		row := RebaseActionRow{
			Repository: r.Name,
			Success:    err == nil,
		}
		if err != nil {
			row.Error = err.Error()
		}
		rows[i] = row
	}

	if jobs == 1 || len(repos) <= 1 {
		for i := range repos {
			do(i, ctx)
		}
		return rows, nil
	}

	sem := semaphore.NewWeighted(int64(jobs))
	g, gctx := errgroup.WithContext(ctx)
	for i := range repos {
		i := i
		if err := sem.Acquire(gctx, 1); err != nil {
			return nil, err
		}
		g.Go(func() error {
			defer sem.Release(1)
			do(i, gctx)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return rows, nil
}

func (rw *RebaseWorkflow) filterRepos(repository string) []wsm.Repository {
	repos := rw.workspace.Repositories
	if repository == "" {
		return repos
	}

	filtered := make([]wsm.Repository, 0, len(repos))
	for _, repo := range repos {
		if repo.Name == repository {
			filtered = append(filtered, repo)
		}
	}
	return filtered
}

func (rw *RebaseWorkflow) rebaseRepository(ctx context.Context, repoName, targetBranch string, interactive, dryRun bool) RebaseResult {
	result := RebaseResult{
		Repository:   repoName,
		Success:      true,
		TargetBranch: targetBranch,
	}

	repoPath := filepath.Join(rw.workspace.Path, repoName)

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		result.Success = false
		result.Error = "repository not found in workspace"
		return result
	}

	currentBranch, err := getCurrentBranch(ctx, repoPath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to get current branch: %v", err)
		return result
	}

	if currentBranch == targetBranch {
		result.Success = true
		result.Error = fmt.Sprintf("already on target branch '%s'", targetBranch)
		return result
	}

	commitsBefore, err := getCommitsAhead(ctx, repoPath, targetBranch)
	if err != nil {
		output.LogWarn(
			fmt.Sprintf("Could not get commits count before rebase for '%s': %v", repoName, err),
			"Could not get commits count before rebase",
			"error", err,
			"repo", repoName,
		)
	}
	result.CommitsBefore = commitsBefore

	if dryRun {
		result.Error = "dry-run mode"
		return result
	}

	if !localBranchExists(ctx, repoPath, targetBranch) {
		if !remoteTrackingBranchExists(ctx, repoPath, targetBranch) {
			result.Success = false
			result.Error = fmt.Sprintf("target branch '%s' not found locally or on remote", targetBranch)
			return result
		}
		if err := fetchBranch(ctx, repoPath, targetBranch); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("target branch '%s' not found locally or on remote", targetBranch)
			return result
		}
	}

	if err := performRebase(ctx, repoPath, targetBranch, interactive); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("rebase failed: %v", err)
		result.Conflicts = hasRebaseConflicts(ctx, repoPath)
		return result
	}

	result.Rebased = true

	commitsAfter, err := getCommitsAhead(ctx, repoPath, targetBranch)
	if err != nil {
		output.LogWarn(
			fmt.Sprintf("Could not get commits count after rebase for '%s': %v", repoName, err),
			"Could not get commits count after rebase",
			"error", err,
			"repo", repoName,
		)
	}
	result.CommitsAfter = commitsAfter

	output.LogInfo(
		fmt.Sprintf("Repository %s rebase completed", repoName),
		"Repository rebase completed",
		"repository", repoName,
		"target", targetBranch,
		"commits_before", result.CommitsBefore,
		"commits_after", result.CommitsAfter,
	)

	return result
}

func getCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func localBranchExists(ctx context.Context, repoPath, branch string) bool {
	branches := wsm.BuildBranchService(ctx)
	exists, err := branches.LocalExists(ctx, repoPath, branchsvc.BranchName(branch))
	return err == nil && exists
}

func remoteTrackingBranchExists(ctx context.Context, repoPath, branch string) bool {
	branches := wsm.BuildBranchService(ctx)
	exists, err := branches.RemoteTrackingExists(ctx, repoPath, branchsvc.DefaultRemoteName, branchsvc.BranchName(branch))
	return err == nil && exists
}

func fetchBranch(ctx context.Context, repoPath, branch string) error {
	// #nosec G204 -- git is invoked with a literal binary and program-derived args; no shell, no user-tainted input.
	cmd := exec.CommandContext(ctx, "git", "fetch", string(branchsvc.DefaultRemoteName), branch+":"+branch)
	cmd.Dir = repoPath
	return cmd.Run()
}

func performRebase(ctx context.Context, repoPath, targetBranch string, interactive bool) error {
	var cmd *exec.Cmd
	// #nosec G204 -- git is invoked with a literal binary and program-derived args; no shell, no user-tainted input.
	if interactive {
		cmd = exec.CommandContext(ctx, "git", "rebase", "-i", targetBranch)
	} else {
		cmd = exec.CommandContext(ctx, "git", "rebase", targetBranch)
	}
	cmd.Dir = repoPath

	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "git rebase failed: %s", string(out))
	}

	return nil
}

func getCommitsAhead(ctx context.Context, repoPath, targetBranch string) (int, error) {
	// #nosec G204 -- git is invoked with a literal binary and program-derived args; no shell, no user-tainted input.
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("HEAD..%s", targetBranch))
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &count); err != nil {
		return 0, err
	}
	return count, nil
}

func hasRebaseConflicts(ctx context.Context, repoPath string) bool {
	state, conflicts, err := wsm.Status(ctx, repoPath)
	if err != nil {
		return false
	}
	if len(conflicts) > 0 {
		return true
	}
	return state == wsm.RebaseStateInProgress || state == wsm.RebaseStateStoppedConflicts
}
