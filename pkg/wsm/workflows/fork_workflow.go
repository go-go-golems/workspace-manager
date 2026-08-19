package workflows

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	branch "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"github.com/pkg/errors"
)

// ForkRequest captures workspace fork options.
type ForkRequest struct {
	NewWorkspaceName    string
	SourceWorkspaceName string
	Branch              string
	BranchPrefix        string
	// BaseBranch is an explicit base/upstream branch to fork from. When set,
	// Plan uses it directly and skips the uniform-branch check, so a fork can
	// proceed even when source repos are on different branches (F1).
	BaseBranch  string
	AgentSource string
	DryRun      bool
}

// ErrBranchDivergence is returned by ForkWorkflow.Plan when source workspace
// repositories are on different branches and no explicit BaseBranch was
// provided. It carries the per-repo branch map and the conventional expected
// branch so the caller (CLI) can prompt the user to choose a base instead of
// hard-failing.
type ErrBranchDivergence struct {
	// Branches maps each repo name to its current branch.
	Branches map[string]string
	// Expected is the conventional branch for the source workspace name
	// (task/<source-name> from BuildWorkspaceBranch), offered as a default.
	Expected string
	// Source is the source workspace name, for messaging.
	Source string
}

// Error implements the error interface.
func (e *ErrBranchDivergence) Error() string {
	distinct := e.DistinctBranches()
	return fmt.Sprintf("repositories in source workspace '%s' are on different branches (%s); pass --base-branch or confirm interactively",
		e.Source, strings.Join(distinct, ", "))
}

// DistinctBranches returns the sorted unique branch names observed across the
// source repositories, for building a prompt or an error message.
func (e *ErrBranchDivergence) DistinctBranches() []string {
	return branch.DistinctBranches(e.Branches)
}

// ForkPlan captures source and derived values for a fork operation.
type ForkPlan struct {
	SourceWorkspace  *wsm.Workspace
	BaseBranch       string
	FinalBranch      string
	RepoNames        []string
	FinalAgentSource string
}

// ForkWorkflow orchestrates workspace fork operations.
type ForkWorkflow struct {
	manager          *wsm.WorkspaceManager
	workspaceContext *wsm.WorkspaceContextService
	checker          *wsm.StatusChecker
}

// NewForkWorkflow creates a fork workflow service.
func NewForkWorkflow() (*ForkWorkflow, error) {
	manager, err := wsm.NewWorkspaceManager()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace manager")
	}

	return &ForkWorkflow{
		manager:          manager,
		workspaceContext: wsm.NewWorkspaceContextService(),
		checker:          wsm.NewStatusChecker(),
	}, nil
}

// Plan resolves source workspace and validates branch assumptions for forking.
func (fw *ForkWorkflow) Plan(ctx context.Context, req ForkRequest) (*ForkPlan, error) {
	sourceWorkspaceName := req.SourceWorkspaceName
	if sourceWorkspaceName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current directory")
		}
		detected, err := fw.workspaceContext.DetectWorkspaceName(cwd)
		if err != nil {
			return nil, errors.Wrap(err, "failed to detect workspace. Use 'workspace-manager fork <new-name> <source-workspace>' or specify --workspace flag")
		}
		sourceWorkspaceName = detected
	}

	sourceWorkspace, err := fw.workspaceContext.LoadWorkspace(sourceWorkspaceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load source workspace '%s'", sourceWorkspaceName)
	}

	status, err := fw.checker.GetWorkspaceStatus(ctx, sourceWorkspace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get source workspace status")
	}
	if len(status.Repositories) == 0 {
		return nil, errors.New("source workspace has no repositories")
	}

	// Resolve the base branch.
	// 1) Explicit override (F1): use it directly and skip the uniformity check.
	var baseBranch string
	if req.BaseBranch != "" {
		baseBranch = req.BaseBranch
	} else {
		// 2) Detect from source repos; require uniformity, else return a typed
		// divergence error so the CLI can prompt instead of hard-failing.
		branches := make(map[string]string, len(status.Repositories))
		for _, rs := range status.Repositories {
			branches[rs.Repository.Name] = rs.CurrentBranch
		}
		distinct := branch.DistinctBranches(branches)
		if len(distinct) == 0 {
			return nil, errors.New("failed to detect base branch from source workspace")
		}
		if len(distinct) == 1 {
			baseBranch = distinct[0]
		} else {
			expected, _ := BuildWorkspaceBranch(sourceWorkspaceName, "", "task")
			return nil, &ErrBranchDivergence{
				Branches: branches,
				Expected: expected,
				Source:   sourceWorkspaceName,
			}
		}
	}

	finalBranch, _ := BuildWorkspaceBranch(req.NewWorkspaceName, req.Branch, req.BranchPrefix)

	repoNames := make([]string, 0, len(sourceWorkspace.Repositories))
	for _, repo := range sourceWorkspace.Repositories {
		repoNames = append(repoNames, repo.Name)
	}

	finalAgentSource := req.AgentSource
	if finalAgentSource == "" {
		finalAgentSource = sourceWorkspace.AgentMD
	}

	return &ForkPlan{
		SourceWorkspace:  sourceWorkspace,
		BaseBranch:       baseBranch,
		FinalBranch:      finalBranch,
		RepoNames:        repoNames,
		FinalAgentSource: finalAgentSource,
	}, nil
}

// Fork executes a fork operation using a validated plan.
func (fw *ForkWorkflow) Fork(ctx context.Context, req ForkRequest) (*wsm.Workspace, *ForkPlan, error) {
	plan, err := fw.Plan(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	workspace, err := fw.manager.CreateWorkspace(
		ctx,
		req.NewWorkspaceName,
		plan.RepoNames,
		plan.FinalBranch,
		plan.BaseBranch,
		plan.FinalAgentSource,
		req.DryRun,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to fork workspace")
	}

	return workspace, plan, nil
}
