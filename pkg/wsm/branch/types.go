package branch

import (
	"os"
	"strings"
)

// BranchName is a typed branch identifier (for example "main" or "feature/foo").
type BranchName string

// RemoteName is a typed remote identifier (for example "origin").
type RemoteName string

const (
	// DefaultRemoteName is used when a request does not specify a remote.
	DefaultRemoteName RemoteName = "origin"
	// DefaultBaseBranch is used when no explicit base branch is configured.
	DefaultBaseBranch BranchName = "main"
)

// ResolveBaseBranch returns the configured base branch with a safe default.
//
// Priority:
// 1. explicit argument
// 2. WSM_BASE_BRANCH environment variable
// 3. DefaultBaseBranch ("main")
func ResolveBaseBranch(explicit string) BranchName {
	if strings.TrimSpace(explicit) != "" {
		return BranchName(strings.TrimSpace(explicit))
	}
	if v := strings.TrimSpace(os.Getenv("WSM_BASE_BRANCH")); v != "" {
		return BranchName(v)
	}
	return DefaultBaseBranch
}

// ResolutionMode defines the operation context for branch planning.
type ResolutionMode int

const (
	ResolutionModeUnspecified ResolutionMode = iota
	ResolutionModeCreateWorktree
	ResolutionModeAddRepository
	ResolutionModeSync
)

func (m ResolutionMode) String() string {
	switch m {
	case ResolutionModeUnspecified:
		return "unspecified"
	case ResolutionModeCreateWorktree:
		return "create-worktree"
	case ResolutionModeAddRepository:
		return "add-repository"
	case ResolutionModeSync:
		return "sync"
	default:
		return "unknown"
	}
}

// ResolutionStrategy captures the selected branch action.
type ResolutionStrategy int

const (
	ResolutionStrategyUnspecified ResolutionStrategy = iota
	ResolutionStrategyUseLocal
	ResolutionStrategyTrackRemote
	ResolutionStrategyCreateFromBase
	ResolutionStrategyCreateFromHead
)

func (s ResolutionStrategy) String() string {
	switch s {
	case ResolutionStrategyUnspecified:
		return "unspecified"
	case ResolutionStrategyUseLocal:
		return "use-local"
	case ResolutionStrategyTrackRemote:
		return "track-remote"
	case ResolutionStrategyCreateFromBase:
		return "create-from-base"
	case ResolutionStrategyCreateFromHead:
		return "create-from-head"
	default:
		return "unknown"
	}
}

// RemoteRefKind defines how RemoteRef should be interpreted.
type RemoteRefKind int

const (
	RemoteRefKindNone RemoteRefKind = iota
	RemoteRefKindRemoteTrackingBranch
)

func (k RemoteRefKind) String() string {
	switch k {
	case RemoteRefKindNone:
		return "none"
	case RemoteRefKindRemoteTrackingBranch:
		return "remote-tracking-branch"
	default:
		return "unknown"
	}
}

// RemoteTrackingRef builds "<remote>/<branch>" using typed domain values.
func RemoteTrackingRef(remote RemoteName, branch BranchName) string {
	if branch == "" {
		return ""
	}
	r := remote
	if r == "" {
		r = DefaultRemoteName
	}
	return string(r) + "/" + string(branch)
}

// BranchSnapshot captures branch state for a repository at a point in time.
type BranchSnapshot struct {
	CurrentBranch          BranchName
	LocalBranches          []BranchName
	RemoteTrackingBranches map[RemoteName][]BranchName
}

// BranchResolutionRequest describes the branch policy query for an operation.
type BranchResolutionRequest struct {
	TargetBranch BranchName
	BaseBranch   BranchName
	Remote       RemoteName
	Mode         ResolutionMode
}

// BranchResolutionPlan is the deterministic result of branch resolution policy.
type BranchResolutionPlan struct {
	Mode                 ResolutionMode
	Strategy             ResolutionStrategy
	Remote               RemoteName
	TargetBranch         BranchName
	LocalExists          bool
	RemoteTrackingExists bool

	RemoteRefKind RemoteRefKind
	RemoteRef     string
	StartPoint    string
}
