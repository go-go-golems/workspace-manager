package wsm

import (
	"time"
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
	IsMerged       bool       `json:"is_merged"`    // True if branch is merged to origin/main
	NeedsRebase    bool       `json:"needs_rebase"` // True if branch needs to be rebased on origin/main
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
