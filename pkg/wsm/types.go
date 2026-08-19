package wsm

import (
	"time"

	branch "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
)

// Repository represents a discovered git repository
type Repository struct {
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	RemoteURL     string    `json:"remote_url"`
	CurrentBranch string    `json:"current_branch"`
	Branches      []string  `json:"branches"`
	Tags          []string  `json:"tags"`
	LastCommit    string    `json:"last_commit"`
	LastUpdated   time.Time `json:"last_updated"`
	Categories    []string  `json:"categories"`
}

// RepositoryRegistry stores discovered repositories
type RepositoryRegistry struct {
	Repositories []Repository `json:"repositories"`
	LastScan     time.Time    `json:"last_scan"`
}

// Workspace represents a multi-repository workspace
type Workspace struct {
	Name         string       `json:"name"`
	Path         string       `json:"path"`
	Repositories []Repository `json:"repositories"`
	Branch       string       `json:"branch"`
	BaseBranch   string       `json:"base_branch"`
	Created      time.Time    `json:"created"`
	GoWorkspace  bool         `json:"go_workspace"`
	AgentMD      string       `json:"agent_md"`
}

// WorkspaceConfig holds workspace management configuration
type WorkspaceConfig struct {
	WorkspaceDir string `json:"workspace_dir"`
	TemplateDir  string `json:"template_dir"`
	RegistryPath string `json:"registry_path"`
}

// BaseComparisonStatus classifies the outcome of resolving the base comparison ref.
// It is a type alias of branch.BaseResolutionStatus so the resolution layer
// (pkg/wsm/branch) remains the single source of truth while wsm exposes a
// stable name for JSON marshaling and the status table.
type BaseComparisonStatus = branch.BaseResolutionStatus

const (
	// BaseResolved means a concrete ref was found and a real comparison ran.
	BaseResolved = branch.BaseResolved
	// BaseUnknown means no usable ref exists (e.g. forked workspace with a
	// local-only base branch that is absent both remotely and locally).
	BaseUnknown = branch.BaseUnknown
	// BaseError means git itself failed while attempting the comparison.
	BaseError = branch.BaseError
)

// RefSource describes how a base comparison ref was resolved. Alias of
// branch.RefSource for a single source of truth.
type RefSource = branch.RefSource

const (
	// RefSourceRemoteTracking means the ref is a remote-tracking branch (<remote>/<branch>).
	RefSourceRemoteTracking = branch.RefSourceRemoteTracking
	// RefSourceLocal means the ref is a local branch (<branch>).
	RefSourceLocal = branch.RefSourceLocal
)

// BaseComparison is the provenance-bearing result of a merge/rebase status
// check. It records which ref was compared against, how it was resolved, and
// why if the comparison could not run. IsMerged/NeedsRebase are meaningful only
// when Status == BaseResolved.
type BaseComparison struct {
	// ConfiguredBase is the base branch name chosen by precedence resolution
	// (per-repo override > workspace base > discovered default > env > main).
	ConfiguredBase string `json:"configured_base"`
	// Remote is the remote used (default "origin").
	Remote string `json:"remote"`
	// ResolvedRef is the concrete git ref compared against (e.g. "origin/main",
	// "task/deploy-dev-indexer"). Empty when Status != BaseResolved.
	ResolvedRef string `json:"resolved_ref"`
	// RefSource is how ResolvedRef was found (remote-tracking | local). Empty
	// when Status != BaseResolved.
	RefSource RefSource `json:"ref_source,omitempty"`
	// Status classifies the outcome (resolved | unknown | error).
	Status BaseComparisonStatus `json:"base_status"`
	// Reason is a human-readable explanation when Status != BaseResolved.
	Reason string `json:"reason,omitempty"`
	// IsMerged is valid only when Status == BaseResolved.
	IsMerged bool `json:"is_merged"`
	// NeedsRebase is valid only when Status == BaseResolved.
	NeedsRebase bool `json:"needs_rebase"`
}

// RepositoryStatus represents the git status of a repository
type RepositoryStatus struct {
	Repository     Repository `json:"repository"`
	HasChanges     bool       `json:"has_changes"`
	StagedFiles    []string   `json:"staged_files"`
	ModifiedFiles  []string   `json:"modified_files"`
	UntrackedFiles []string   `json:"untracked_files"`
	Ahead          int        `json:"ahead"`
	Behind         int        `json:"behind"`
	CurrentBranch  string     `json:"current_branch"`
	HasConflicts   bool       `json:"has_conflicts"`
	// IsMerged/NeedsRebase are mirrors of Base.IsMerged/NeedsRebase kept for
	// JSON compatibility with existing consumers. New consumers should read
	// Base for provenance.
	IsMerged    bool           `json:"is_merged"`
	NeedsRebase bool           `json:"needs_rebase"`
	Base        BaseComparison `json:"base"`
}

// WorkspaceStatus represents the overall status of a workspace
type WorkspaceStatus struct {
	Workspace    Workspace          `json:"workspace"`
	Repositories []RepositoryStatus `json:"repositories"`
	Overall      string             `json:"overall"`
}

// WorktreeInfo tracks information about a created worktree for rollback purposes
type WorktreeInfo struct {
	Repository Repository `json:"repository"`
	TargetPath string     `json:"target_path"`
	Branch     string     `json:"branch"`
}
