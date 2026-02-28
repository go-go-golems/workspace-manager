package branch

import "context"

// Service centralizes branch state queries and branch decision policy.
type Service interface {
	Snapshot(ctx context.Context, repoPath string) (*BranchSnapshot, error)
	Resolve(ctx context.Context, repoPath string, req BranchResolutionRequest) (*BranchResolutionPlan, error)

	LocalExists(ctx context.Context, repoPath string, branch BranchName) (bool, error)
	RemoteTrackingExists(ctx context.Context, repoPath string, remote RemoteName, branch BranchName) (bool, error)
	ListLocal(ctx context.Context, repoPath string) ([]BranchName, error)
	ListRemoteTracking(ctx context.Context, repoPath string, remote RemoteName) ([]BranchName, error)
}
