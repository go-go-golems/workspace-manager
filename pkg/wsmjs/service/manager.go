package service

import (
	"context"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
)

// ManagerOptions configures default manager behavior for JS callers.
type ManagerOptions struct {
	DefaultJobs int
}

// DiscoverInput mirrors workflows.DiscoverRequest with JS-facing naming.
type DiscoverInput struct {
	Paths     []string
	Recursive bool
	MaxDepth  int
}

// CreateWorkspaceInput mirrors workflows.CreateRequest with JS-facing naming.
type CreateWorkspaceInput struct {
	Name         string
	Repos        []string
	Branch       string
	BranchPrefix string
	BaseBranch   string
	AgentSource  string
	DryRun       bool
}

// StatusInput mirrors workflows.StatusRequest with JS-facing naming.
type StatusInput struct {
	WorkspaceName string
	Jobs          int
}

// ListRepositoriesInput controls registry repository listing.
type ListRepositoriesInput struct {
	Tags []string
}

// Manager is a package-level facade exposing workflow-backed operations.
type Manager struct {
	defaultJobs int
}

// NewManager creates a new workflow facade.
func NewManager(opts ManagerOptions) *Manager {
	jobs := opts.DefaultJobs
	if jobs <= 0 {
		jobs = 8
	}
	return &Manager{defaultJobs: jobs}
}

// Discover runs repository discovery via workflow facade.
func (m *Manager) Discover(ctx context.Context, in DiscoverInput) (*workflows.DiscoverResult, error) {
	workflow, err := workflows.NewDiscoverWorkflow()
	if err != nil {
		return nil, err
	}

	recursive := in.Recursive
	if !in.Recursive && len(in.Paths) == 0 && in.MaxDepth == 0 {
		// Keep default behavior aligned with CLI defaults for empty input.
		recursive = true
	}
	maxDepth := in.MaxDepth
	if maxDepth == 0 {
		maxDepth = 3
	}

	return workflow.Discover(ctx, workflows.DiscoverRequest{
		Paths:     in.Paths,
		Recursive: recursive,
		MaxDepth:  maxDepth,
	})
}

// CreateWorkspace creates a workspace via the create workflow.
func (m *Manager) CreateWorkspace(ctx context.Context, in CreateWorkspaceInput) (*workflows.CreateResult, error) {
	workflow, err := workflows.NewCreateWorkflow()
	if err != nil {
		return nil, err
	}

	return workflow.Create(ctx, workflows.CreateRequest{
		Name:         in.Name,
		Repos:        in.Repos,
		Branch:       in.Branch,
		BranchPrefix: in.BranchPrefix,
		BaseBranch:   in.BaseBranch,
		AgentSource:  in.AgentSource,
		DryRun:       in.DryRun,
	})
}

// Status retrieves workspace status via status workflow.
func (m *Manager) Status(ctx context.Context, in StatusInput) (*wsm.WorkspaceStatus, error) {
	workflow := workflows.NewStatusWorkflow()
	jobs := in.Jobs
	if jobs <= 0 {
		jobs = m.defaultJobs
	}
	return workflow.GetStatus(ctx, workflows.StatusRequest{
		WorkspaceName: in.WorkspaceName,
		Jobs:          jobs,
	})
}

// ListWorkspaces returns all known workspaces.
func (m *Manager) ListWorkspaces(_ context.Context) ([]wsm.Workspace, error) {
	workflow, err := workflows.NewListWorkflow()
	if err != nil {
		return nil, err
	}
	return workflow.ListWorkspaces()
}

// ListRepositories returns discovered repositories.
func (m *Manager) ListRepositories(_ context.Context, in ListRepositoriesInput) ([]wsm.Repository, error) {
	workflow, err := workflows.NewListWorkflow()
	if err != nil {
		return nil, err
	}
	return workflow.ListRepositories(in.Tags)
}
