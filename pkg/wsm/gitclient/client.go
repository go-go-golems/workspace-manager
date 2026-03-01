package gitclient

import (
	"context"
)

// RepositoryHandle abstracts a repository reference for a given backend.
type RepositoryHandle interface {
	Path() string
}

// FileChange represents a change to a file as reported by Status.
type FileChange struct {
	Path   string
	Staged bool
	// Status: one of "A", "M", "D", "R", "C", "?"
	Status string
}

// Status represents working tree status for a repository.
type Status struct {
	CurrentBranch  string
	ModifiedFiles  []string
	StagedFiles    []string
	UntrackedFiles []string
}

// WorktreeAddOptions controls worktree add behavior.
type WorktreeAddOptions struct {
	BaseRef           string // base ref to create branch from, optional
	Overwrite         bool   // use -B semantics when true
	RemoteBranch      string // origin/<branch> when creating from remote
	UseExistingBranch bool   // when true, checkout existing branch without creating a new one
}

// WorktreeInfo describes an existing worktree.
type WorktreeInfo struct {
	Path   string
	Branch string
}

// GitClient defines repository-level git operations.
type GitClient interface {
	Open(ctx context.Context, repoPath string) (RepositoryHandle, error)

	// Introspection
	CurrentBranch(ctx context.Context, repo RepositoryHandle) (string, error)
	RemoteURL(ctx context.Context, repo RepositoryHandle, remote string) (string, error)
	// LocalBranchExists reports whether refs/heads/<branch> exists.
	LocalBranchExists(ctx context.Context, repo RepositoryHandle, branch string) (bool, error)
	// ListLocalBranches returns refs/heads entries.
	ListLocalBranches(ctx context.Context, repo RepositoryHandle) ([]string, error)
	// ListRemoteTrackingBranches returns branch names for refs/remotes/<remote>/*.
	ListRemoteTrackingBranches(ctx context.Context, repo RepositoryHandle, remote string) ([]string, error)
	// RemoteTrackingBranchExists reports whether refs/remotes/<remote>/<branch> exists.
	RemoteTrackingBranchExists(ctx context.Context, repo RepositoryHandle, remote string, branch string) (bool, error)
	ListTags(ctx context.Context, repo RepositoryHandle) ([]string, error)
	LastCommit(ctx context.Context, repo RepositoryHandle) (string, error)

	// Working tree
	Status(ctx context.Context, repo RepositoryHandle) (Status, error)
	Add(ctx context.Context, repo RepositoryHandle, path string) error
	Reset(ctx context.Context, repo RepositoryHandle, path string) error
	Commit(ctx context.Context, repo RepositoryHandle, msg string) error
	Diff(ctx context.Context, repo RepositoryHandle, staged bool) (string, error)

	// Sync
	Fetch(ctx context.Context, repo RepositoryHandle, remote string) error
	Push(ctx context.Context, repo RepositoryHandle, remote string) error
	// If upstream is empty, implementation should use the current branch upstream if configured.
	AheadBehind(ctx context.Context, repo RepositoryHandle, upstream string) (ahead int, behind int, err error)

	// Branches
	CreateBranch(ctx context.Context, repo RepositoryHandle, name string, track bool, baseRef string) error
	CheckoutBranch(ctx context.Context, repo RepositoryHandle, name string, create bool, force bool) error
}

// WorktreeManager defines operations for git worktrees.
type WorktreeManager interface {
	Add(ctx context.Context, repoPath string, branch string, targetPath string, opts WorktreeAddOptions) error
	Remove(ctx context.Context, repoPath string, targetPath string, force bool) error
	List(ctx context.Context, repoPath string) ([]WorktreeInfo, error)
}
