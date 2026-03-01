package service

import (
	"context"
	"os"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/pkg/errors"
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

// InfoInput resolves full workspace info or a specific field.
type InfoInput struct {
	WorkspaceName string
	Field         string
}

// InfoResult contains workspace information and optional single-field extraction.
type InfoResult struct {
	Workspace *wsm.Workspace `json:"workspace"`
	Field     string         `json:"field,omitempty"`
	Value     string         `json:"value,omitempty"`
	HasField  bool           `json:"hasField"`
}

// AddRepositoryInput adds one repository to an existing workspace.
type AddRepositoryInput struct {
	WorkspaceName string
	RepoName      string
	Branch        string
	Force         bool
}

// AddRepositoryResult summarizes add operation inputs.
type AddRepositoryResult struct {
	WorkspaceName string `json:"workspaceName"`
	RepoName      string `json:"repoName"`
	Branch        string `json:"branch"`
	Force         bool   `json:"force"`
	Status        string `json:"status"`
}

// RemoveRepositoryInput removes one repository from an existing workspace.
type RemoveRepositoryInput struct {
	WorkspaceName string
	RepoName      string
	Force         bool
	RemoveFiles   bool
}

// RemoveRepositoryResult summarizes remove operation inputs.
type RemoveRepositoryResult struct {
	WorkspaceName string `json:"workspaceName"`
	RepoName      string `json:"repoName"`
	Force         bool   `json:"force"`
	RemoveFiles   bool   `json:"removeFiles"`
	Status        string `json:"status"`
}

// DeleteWorkspaceInput deletes a workspace and optionally removes files/worktrees.
type DeleteWorkspaceInput struct {
	WorkspaceName  string
	RemoveFiles    bool
	ForceWorktrees bool
}

// DeleteWorkspaceResult summarizes delete operation inputs.
type DeleteWorkspaceResult struct {
	WorkspaceName  string `json:"workspaceName"`
	RemoveFiles    bool   `json:"removeFiles"`
	ForceWorktrees bool   `json:"forceWorktrees"`
	Status         string `json:"status"`
}

// ForkWorkspaceInput captures workspace fork options.
type ForkWorkspaceInput struct {
	NewWorkspaceName    string
	SourceWorkspaceName string
	Branch              string
	BranchPrefix        string
	AgentSource         string
	DryRun              bool
}

// ForkWorkspaceResult returns created workspace and computed fork plan details.
type ForkWorkspaceResult struct {
	Workspace *wsm.Workspace      `json:"workspace"`
	Plan      *workflows.ForkPlan `json:"plan"`
}

// MergeWorkspaceInput captures merge execution options.
type MergeWorkspaceInput struct {
	WorkspaceName string
	DryRun        bool
	Force         bool
	KeepWorkspace bool
}

// MergeWorkspaceResult summarizes merge execution options.
type MergeWorkspaceResult struct {
	WorkspaceName string `json:"workspaceName"`
	DryRun        bool   `json:"dryRun"`
	Force         bool   `json:"force"`
	KeepWorkspace bool   `json:"keepWorkspace"`
	Status        string `json:"status"`
}

// CommitInput captures JS-facing commit options.
type CommitInput struct {
	WorkspaceName   string
	Message         string
	Template        string
	AddAll          bool
	Push            bool
	DryRun          bool
	SelectedChanges map[string][]wsm.FileChange
}

// CommitResult summarizes commit execution details.
type CommitResult struct {
	WorkspaceName   string                      `json:"workspaceName"`
	Message         string                      `json:"message"`
	AddAll          bool                        `json:"addAll"`
	Push            bool                        `json:"push"`
	DryRun          bool                        `json:"dryRun"`
	SelectedChanges map[string][]wsm.FileChange `json:"selectedChanges"`
	Status          string                      `json:"status"`
}

// DiffInput captures JS-facing diff options.
type DiffInput struct {
	WorkspaceName string
	Staged        bool
	Repo          string
	Jobs          int
}

// DiffResult contains combined workspace diff output.
type DiffResult struct {
	WorkspaceName string `json:"workspaceName"`
	Diff          string `json:"diff"`
	Staged        bool   `json:"staged"`
	Repo          string `json:"repo"`
	Jobs          int    `json:"jobs"`
	HasChanges    bool   `json:"hasChanges"`
}

// LogInput captures JS-facing log options.
type LogInput struct {
	WorkspaceName string
	Since         string
	Oneline       bool
	Limit         int
}

// LogResult contains per-repository log output.
type LogResult struct {
	WorkspaceName string            `json:"workspaceName"`
	Since         string            `json:"since"`
	Oneline       bool              `json:"oneline"`
	Limit         int               `json:"limit"`
	Logs          map[string]string `json:"logs"`
}

// BranchCreateInput captures branch creation options.
type BranchCreateInput struct {
	WorkspaceName string
	Repo          string
	BranchName    string
	Track         bool
}

// BranchCreateResult summarizes branch creation behavior.
type BranchCreateResult struct {
	WorkspaceName string                      `json:"workspaceName"`
	Repo          string                      `json:"repo"`
	BranchName    string                      `json:"branchName"`
	Track         bool                        `json:"track"`
	Results       []wsm.BranchOperationResult `json:"results"`
}

// BranchSwitchInput captures branch switch options.
type BranchSwitchInput struct {
	WorkspaceName string
	Repo          string
	BranchName    string
}

// BranchSwitchResult summarizes branch switch behavior.
type BranchSwitchResult struct {
	WorkspaceName string                      `json:"workspaceName"`
	Repo          string                      `json:"repo"`
	BranchName    string                      `json:"branchName"`
	Results       []wsm.BranchOperationResult `json:"results"`
}

// BranchListInput captures branch listing options.
type BranchListInput struct {
	WorkspaceName string
	Repo          string
	Jobs          int
}

// BranchListEntry is one branch/status row for a repository.
type BranchListEntry struct {
	Repository    string `json:"repository"`
	CurrentBranch string `json:"currentBranch"`
	StatusSymbol  string `json:"statusSymbol"`
	HasChanges    bool   `json:"hasChanges"`
	HasConflicts  bool   `json:"hasConflicts"`
	Error         string `json:"error"`
}

// BranchListResult contains all branch rows for a workspace.
type BranchListResult struct {
	WorkspaceName string            `json:"workspaceName"`
	Repo          string            `json:"repo"`
	Jobs          int               `json:"jobs"`
	Entries       []BranchListEntry `json:"entries"`
}

// RebaseRunInput captures rebase run options.
type RebaseRunInput struct {
	WorkspaceName string
	Repository    string
	TargetBranch  string
	Interactive   bool
	DryRun        bool
	Jobs          int
	Manual        bool
}

// RebaseRunResult contains rebase execution rows or manual command plan.
type RebaseRunResult struct {
	WorkspaceName string                   `json:"workspaceName"`
	Repository    string                   `json:"repository"`
	TargetBranch  string                   `json:"targetBranch"`
	Interactive   bool                     `json:"interactive"`
	DryRun        bool                     `json:"dryRun"`
	Jobs          int                      `json:"jobs"`
	Manual        bool                     `json:"manual"`
	Commands      []string                 `json:"commands"`
	Results       []workflows.RebaseResult `json:"results"`
}

// RebaseStatusInput captures rebase status options.
type RebaseStatusInput struct {
	WorkspaceName string
	Repository    string
	Jobs          int
}

// RebaseStatusResult returns rebase status rows across repositories.
type RebaseStatusResult struct {
	WorkspaceName string                      `json:"workspaceName"`
	Jobs          int                         `json:"jobs"`
	Rows          []workflows.RebaseStatusRow `json:"rows"`
}

// RebaseActionInput captures rebase continue/abort options.
type RebaseActionInput struct {
	WorkspaceName string
	Repository    string
	Jobs          int
}

// RebaseActionResult returns rows for continue/abort actions.
type RebaseActionResult struct {
	WorkspaceName string                      `json:"workspaceName"`
	Mode          string                      `json:"mode"`
	Jobs          int                         `json:"jobs"`
	Rows          []workflows.RebaseActionRow `json:"rows"`
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
	return workflow.GetStatus(ctx, workflows.StatusRequest{
		WorkspaceName: in.WorkspaceName,
		Jobs:          m.normalizeJobs(in.Jobs),
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

// LoadWorkspace loads a workspace by name.
func (m *Manager) LoadWorkspace(_ context.Context, workspaceName string) (*wsm.Workspace, error) {
	if workspaceName == "" {
		return nil, errors.New("workspaceName is required")
	}
	workspaceContext := wsm.NewWorkspaceContextService()
	workspace, err := workspaceContext.LoadWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}
	return workspace, nil
}

// Info resolves full workspace info or one field.
func (m *Manager) Info(_ context.Context, in InfoInput) (*InfoResult, error) {
	workflow := workflows.NewInfoWorkflow()
	workspace, err := workflow.ResolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}

	if in.Field != "" {
		value, err := workflow.FieldValue(workspace, in.Field)
		if err != nil {
			return nil, err
		}
		return &InfoResult{
			Workspace: workspace,
			Field:     strings.ToLower(in.Field),
			Value:     value,
			HasField:  true,
		}, nil
	}

	return &InfoResult{Workspace: workspace, HasField: false}, nil
}

// AddRepository adds a repository to an existing workspace.
func (m *Manager) AddRepository(ctx context.Context, in AddRepositoryInput) (*AddRepositoryResult, error) {
	if in.WorkspaceName == "" {
		return nil, errors.New("workspaceName is required")
	}
	if in.RepoName == "" {
		return nil, errors.New("repoName is required")
	}

	wm, err := wsm.NewWorkspaceManager()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace manager")
	}
	if err := wm.AddRepositoryToWorkspace(ctx, in.WorkspaceName, in.RepoName, in.Branch, in.Force); err != nil {
		return nil, err
	}

	return &AddRepositoryResult{
		WorkspaceName: in.WorkspaceName,
		RepoName:      in.RepoName,
		Branch:        in.Branch,
		Force:         in.Force,
		Status:        "added",
	}, nil
}

// RemoveRepository removes a repository from an existing workspace.
func (m *Manager) RemoveRepository(ctx context.Context, in RemoveRepositoryInput) (*RemoveRepositoryResult, error) {
	if in.WorkspaceName == "" {
		return nil, errors.New("workspaceName is required")
	}
	if in.RepoName == "" {
		return nil, errors.New("repoName is required")
	}

	wm, err := wsm.NewWorkspaceManager()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace manager")
	}
	if err := wm.RemoveRepositoryFromWorkspace(ctx, in.WorkspaceName, in.RepoName, in.Force, in.RemoveFiles); err != nil {
		return nil, err
	}

	return &RemoveRepositoryResult{
		WorkspaceName: in.WorkspaceName,
		RepoName:      in.RepoName,
		Force:         in.Force,
		RemoveFiles:   in.RemoveFiles,
		Status:        "removed",
	}, nil
}

// DeleteWorkspace deletes a workspace.
func (m *Manager) DeleteWorkspace(ctx context.Context, in DeleteWorkspaceInput) (*DeleteWorkspaceResult, error) {
	if in.WorkspaceName == "" {
		return nil, errors.New("workspaceName is required")
	}

	workflow, err := workflows.NewDeleteWorkflow()
	if err != nil {
		return nil, err
	}
	if err := workflow.Delete(ctx, in.WorkspaceName, in.RemoveFiles, in.ForceWorktrees); err != nil {
		return nil, err
	}

	return &DeleteWorkspaceResult{
		WorkspaceName:  in.WorkspaceName,
		RemoveFiles:    in.RemoveFiles,
		ForceWorktrees: in.ForceWorktrees,
		Status:         "deleted",
	}, nil
}

// ForkWorkspace creates a new workspace by forking a source workspace.
func (m *Manager) ForkWorkspace(ctx context.Context, in ForkWorkspaceInput) (*ForkWorkspaceResult, error) {
	if in.NewWorkspaceName == "" {
		return nil, errors.New("newWorkspaceName is required")
	}

	workflow, err := workflows.NewForkWorkflow()
	if err != nil {
		return nil, err
	}

	workspace, plan, err := workflow.Fork(ctx, workflows.ForkRequest{
		NewWorkspaceName:    in.NewWorkspaceName,
		SourceWorkspaceName: in.SourceWorkspaceName,
		Branch:              in.Branch,
		BranchPrefix:        in.BranchPrefix,
		AgentSource:         in.AgentSource,
		DryRun:              in.DryRun,
	})
	if err != nil {
		return nil, err
	}

	return &ForkWorkspaceResult{Workspace: workspace, Plan: plan}, nil
}

// MergeWorkspace merges a forked workspace into its base branch.
func (m *Manager) MergeWorkspace(ctx context.Context, in MergeWorkspaceInput) (*MergeWorkspaceResult, error) {
	if in.WorkspaceName == "" {
		return nil, errors.New("workspaceName is required")
	}

	workflow := workflows.NewMergeWorkflow()
	if err := workflow.Execute(ctx, workflows.MergeRequest{
		WorkspaceName: in.WorkspaceName,
		DryRun:        in.DryRun,
		Force:         in.Force,
		KeepWorkspace: in.KeepWorkspace,
	}); err != nil {
		return nil, err
	}

	return &MergeWorkspaceResult{
		WorkspaceName: in.WorkspaceName,
		DryRun:        in.DryRun,
		Force:         in.Force,
		KeepWorkspace: in.KeepWorkspace,
		Status:        "merged",
	}, nil
}

// Commit commits selected or all detected changes across workspace repositories.
func (m *Manager) Commit(ctx context.Context, in CommitInput) (*CommitResult, error) {
	workspace, err := m.resolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}

	gitOps := wsm.NewGitOperations(workspace)
	allChanges, err := gitOps.GetWorkspaceChanges(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace changes")
	}

	message := strings.TrimSpace(in.Message)
	if message == "" && in.Template != "" {
		message = workflows.ResolveCommitTemplate(in.Template)
	}

	if len(allChanges) == 0 {
		return &CommitResult{
			WorkspaceName: workspace.Name,
			Message:       message,
			AddAll:        in.AddAll,
			Push:          in.Push,
			DryRun:        in.DryRun,
			Status:        "no_changes",
		}, nil
	}

	if message == "" {
		return nil, errors.New("commit message is required")
	}

	selectedChanges := in.SelectedChanges
	if len(selectedChanges) == 0 {
		selectedChanges = allChanges
	}

	if len(selectedChanges) == 0 {
		return &CommitResult{
			WorkspaceName: workspace.Name,
			Message:       message,
			AddAll:        in.AddAll,
			Push:          in.Push,
			DryRun:        in.DryRun,
			Status:        "no_selection",
		}, nil
	}

	if err := gitOps.CommitChanges(ctx, &wsm.CommitOperation{
		Message: message,
		Files:   selectedChanges,
		DryRun:  in.DryRun,
		AddAll:  in.AddAll,
		Push:    in.Push,
	}); err != nil {
		return nil, errors.Wrap(err, "commit failed")
	}

	return &CommitResult{
		WorkspaceName:   workspace.Name,
		Message:         message,
		AddAll:          in.AddAll,
		Push:            in.Push,
		DryRun:          in.DryRun,
		SelectedChanges: selectedChanges,
		Status:          "executed",
	}, nil
}

// Diff retrieves workspace diff output.
func (m *Manager) Diff(ctx context.Context, in DiffInput) (*DiffResult, error) {
	workspace, err := m.resolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}

	gitOps := wsm.NewGitOperations(workspace)
	diff, err := gitOps.GetDiffWithOptions(ctx, in.Staged, in.Repo, wsm.DiffOptions{MaxJobs: m.normalizeJobs(in.Jobs)})
	if err != nil {
		return nil, err
	}

	noChanges := diff == "" || diff == "No changes found in workspace."
	return &DiffResult{
		WorkspaceName: workspace.Name,
		Diff:          diff,
		Staged:        in.Staged,
		Repo:          in.Repo,
		Jobs:          m.normalizeJobs(in.Jobs),
		HasChanges:    !noChanges,
	}, nil
}

// Log retrieves per-repository commit history for a workspace.
func (m *Manager) Log(ctx context.Context, in LogInput) (*LogResult, error) {
	workspace, err := m.resolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}

	historyOps := wsm.NewHistoryOperations(workspace)
	logs, err := historyOps.GetWorkspaceLog(ctx, in.Since, in.Oneline, in.Limit)
	if err != nil {
		return nil, err
	}

	return &LogResult{
		WorkspaceName: workspace.Name,
		Since:         in.Since,
		Oneline:       in.Oneline,
		Limit:         in.Limit,
		Logs:          logs,
	}, nil
}

// BranchCreate creates a branch across repositories.
func (m *Manager) BranchCreate(ctx context.Context, in BranchCreateInput) (*BranchCreateResult, error) {
	if strings.TrimSpace(in.BranchName) == "" {
		return nil, errors.New("branchName is required")
	}

	workspace, err := m.resolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}
	workspace, err = filterWorkspaceRepositories(workspace, in.Repo)
	if err != nil {
		return nil, err
	}

	branchOps := wsm.NewBranchOperations(workspace)
	results, err := branchOps.CreateBranch(ctx, in.BranchName, in.Track)
	if err != nil {
		return nil, err
	}

	return &BranchCreateResult{
		WorkspaceName: workspace.Name,
		Repo:          in.Repo,
		BranchName:    in.BranchName,
		Track:         in.Track,
		Results:       results,
	}, nil
}

// BranchSwitch switches branch across repositories.
func (m *Manager) BranchSwitch(ctx context.Context, in BranchSwitchInput) (*BranchSwitchResult, error) {
	if strings.TrimSpace(in.BranchName) == "" {
		return nil, errors.New("branchName is required")
	}

	workspace, err := m.resolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}
	workspace, err = filterWorkspaceRepositories(workspace, in.Repo)
	if err != nil {
		return nil, err
	}

	branchOps := wsm.NewBranchOperations(workspace)
	results, err := branchOps.SwitchBranch(ctx, in.BranchName)
	if err != nil {
		return nil, err
	}

	return &BranchSwitchResult{
		WorkspaceName: workspace.Name,
		Repo:          in.Repo,
		BranchName:    in.BranchName,
		Results:       results,
	}, nil
}

// BranchList returns current branch/status rows across repositories.
func (m *Manager) BranchList(ctx context.Context, in BranchListInput) (*BranchListResult, error) {
	workspace, err := m.resolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}

	statusWorkflow := workflows.NewStatusWorkflow()
	status, err := statusWorkflow.GetStatus(ctx, workflows.StatusRequest{
		WorkspaceName: workspace.Name,
		Jobs:          m.normalizeJobs(in.Jobs),
	})
	if err != nil {
		return nil, err
	}

	entries := make([]BranchListEntry, 0, len(status.Repositories))
	for _, repoStatus := range status.Repositories {
		if in.Repo != "" && repoStatus.Repository.Name != in.Repo {
			continue
		}
		symbol := "✅"
		if repoStatus.HasChanges {
			symbol = "🔄"
		}
		if repoStatus.HasConflicts {
			symbol = "⚠️"
		}
		entries = append(entries, BranchListEntry{
			Repository:    repoStatus.Repository.Name,
			CurrentBranch: repoStatus.CurrentBranch,
			StatusSymbol:  symbol,
			HasChanges:    repoStatus.HasChanges,
			HasConflicts:  repoStatus.HasConflicts,
		})
	}

	if in.Repo != "" && len(entries) == 0 {
		return nil, errors.Errorf("repository '%s' not found in workspace '%s'", in.Repo, workspace.Name)
	}

	return &BranchListResult{
		WorkspaceName: workspace.Name,
		Repo:          in.Repo,
		Jobs:          m.normalizeJobs(in.Jobs),
		Entries:       entries,
	}, nil
}

// RebaseRun runs workspace rebase or returns manual command plan.
func (m *Manager) RebaseRun(ctx context.Context, in RebaseRunInput) (*RebaseRunResult, error) {
	workspace, err := m.resolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}

	workflow := workflows.NewRebaseWorkflow(workspace)
	targetBranch := in.TargetBranch
	if targetBranch == "" {
		targetBranch = "main"
	}

	result := &RebaseRunResult{
		WorkspaceName: workspace.Name,
		Repository:    in.Repository,
		TargetBranch:  targetBranch,
		Interactive:   in.Interactive,
		DryRun:        in.DryRun,
		Jobs:          m.normalizeJobs(in.Jobs),
		Manual:        in.Manual,
	}

	if in.Manual {
		result.Commands = workflow.ManualPlan(in.Repository, targetBranch)
		return result, nil
	}

	rows, err := workflow.Rebase(ctx, workflows.RebaseRequest{
		Repository:   in.Repository,
		TargetBranch: targetBranch,
		Interactive:  in.Interactive,
		DryRun:       in.DryRun,
		Jobs:         m.normalizeJobs(in.Jobs),
	})
	if err != nil {
		return nil, err
	}
	result.Results = rows
	return result, nil
}

// RebaseStatus returns rebase status rows across repositories.
func (m *Manager) RebaseStatus(ctx context.Context, in RebaseStatusInput) (*RebaseStatusResult, error) {
	workspace, err := m.resolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}

	workflow := workflows.NewRebaseWorkflow(workspace)
	rows, err := workflow.Status(ctx, in.Repository, m.normalizeJobs(in.Jobs))
	if err != nil {
		return nil, err
	}

	return &RebaseStatusResult{
		WorkspaceName: workspace.Name,
		Jobs:          m.normalizeJobs(in.Jobs),
		Rows:          rows,
	}, nil
}

// RebaseContinue continues rebases across repositories.
func (m *Manager) RebaseContinue(ctx context.Context, in RebaseActionInput) (*RebaseActionResult, error) {
	workspace, err := m.resolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}

	workflow := workflows.NewRebaseWorkflow(workspace)
	rows, err := workflow.Continue(ctx, in.Repository, m.normalizeJobs(in.Jobs))
	if err != nil {
		return nil, err
	}

	return &RebaseActionResult{
		WorkspaceName: workspace.Name,
		Mode:          "continue",
		Jobs:          m.normalizeJobs(in.Jobs),
		Rows:          rows,
	}, nil
}

// RebaseAbort aborts rebases across repositories.
func (m *Manager) RebaseAbort(ctx context.Context, in RebaseActionInput) (*RebaseActionResult, error) {
	workspace, err := m.resolveWorkspace(in.WorkspaceName)
	if err != nil {
		return nil, err
	}

	workflow := workflows.NewRebaseWorkflow(workspace)
	rows, err := workflow.Abort(ctx, in.Repository, m.normalizeJobs(in.Jobs))
	if err != nil {
		return nil, err
	}

	return &RebaseActionResult{
		WorkspaceName: workspace.Name,
		Mode:          "abort",
		Jobs:          m.normalizeJobs(in.Jobs),
		Rows:          rows,
	}, nil
}

func (m *Manager) normalizeJobs(jobs int) int {
	if jobs > 0 {
		return jobs
	}
	return m.defaultJobs
}

func (m *Manager) resolveWorkspace(workspaceName string) (*wsm.Workspace, error) {
	workspaceContext := wsm.NewWorkspaceContextService()
	if strings.TrimSpace(workspaceName) != "" {
		workspace, err := workspaceContext.LoadWorkspace(workspaceName)
		if err != nil {
			return nil, err
		}
		return workspace, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current directory")
	}

	workspace, err := workspaceContext.DetectCurrentWorkspace(cwd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect current workspace")
	}
	return workspace, nil
}

func filterWorkspaceRepositories(workspace *wsm.Workspace, repoName string) (*wsm.Workspace, error) {
	if strings.TrimSpace(repoName) == "" {
		return workspace, nil
	}

	filtered := *workspace
	filtered.Repositories = make([]wsm.Repository, 0, len(workspace.Repositories))
	for _, repo := range workspace.Repositories {
		if repo.Name == repoName {
			filtered.Repositories = append(filtered.Repositories, repo)
		}
	}
	if len(filtered.Repositories) == 0 {
		return nil, errors.Errorf("repository '%s' not found in workspace '%s'", repoName, workspace.Name)
	}
	return &filtered, nil
}
