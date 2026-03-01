package branch

import "testing"

func TestResolveFromState(t *testing.T) {
	tests := []struct {
		name          string
		req           BranchResolutionRequest
		local         bool
		remote        bool
		wantStrategy  ResolutionStrategy
		wantRemoteRef string
		wantRefKind   RemoteRefKind
		wantStart     string
		wantErr       bool
	}{
		{
			name:    "invalid mode",
			req:     BranchResolutionRequest{TargetBranch: "feature", Mode: ResolutionModeUnspecified},
			wantErr: true,
		},
		{
			name:    "empty target branch",
			req:     BranchResolutionRequest{Mode: ResolutionModeCreateWorktree},
			wantErr: true,
		},
		{
			name:         "use local",
			req:          BranchResolutionRequest{TargetBranch: "feature", Mode: ResolutionModeCreateWorktree},
			local:        true,
			wantStrategy: ResolutionStrategyUseLocal,
			wantRefKind:  RemoteRefKindNone,
		},
		{
			name:          "track remote",
			req:           BranchResolutionRequest{TargetBranch: "feature", Mode: ResolutionModeCreateWorktree, Remote: "upstream"},
			remote:        true,
			wantStrategy:  ResolutionStrategyTrackRemote,
			wantRemoteRef: "upstream/feature",
			wantRefKind:   RemoteRefKindRemoteTrackingBranch,
		},
		{
			name:         "create from base",
			req:          BranchResolutionRequest{TargetBranch: "feature", BaseBranch: "main", Mode: ResolutionModeAddRepository},
			wantStrategy: ResolutionStrategyCreateFromBase,
			wantStart:    "main",
			wantRefKind:  RemoteRefKindNone,
		},
		{
			name:         "create from head",
			req:          BranchResolutionRequest{TargetBranch: "feature", Mode: ResolutionModeSync},
			wantStrategy: ResolutionStrategyCreateFromHead,
			wantRefKind:  RemoteRefKindNone,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := resolveFromState(tc.req, DefaultRemoteName, tc.local, tc.remote)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if plan.Strategy != tc.wantStrategy {
				t.Fatalf("expected strategy %v, got %v", tc.wantStrategy, plan.Strategy)
			}
			if plan.RemoteRef != tc.wantRemoteRef {
				t.Fatalf("expected remote ref %q, got %q", tc.wantRemoteRef, plan.RemoteRef)
			}
			if plan.RemoteRefKind != tc.wantRefKind {
				t.Fatalf("expected remote ref kind %v, got %v", tc.wantRefKind, plan.RemoteRefKind)
			}
			if plan.StartPoint != tc.wantStart {
				t.Fatalf("expected start point %q, got %q", tc.wantStart, plan.StartPoint)
			}
		})
	}
}
