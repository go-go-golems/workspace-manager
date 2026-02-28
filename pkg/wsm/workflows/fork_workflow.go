package workflows

import (
	"context"
	"os"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
)

// ForkRequest captures workspace fork options.
type ForkRequest struct {
	NewWorkspaceName    string
	SourceWorkspaceName string
	Branch              string
	BranchPrefix        string
	AgentSource         string
	DryRun              bool
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

	baseBranch := status.Repositories[0].CurrentBranch
	if baseBranch == "" {
		return nil, errors.New("failed to detect base branch from source workspace")
	}

	for _, repoStatus := range status.Repositories {
		if repoStatus.CurrentBranch != baseBranch {
			return nil, errors.Errorf("repositories in source workspace are on different branches: %s is on %s, but expected %s",
				repoStatus.Repository.Name, repoStatus.CurrentBranch, baseBranch)
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
