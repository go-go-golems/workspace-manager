# Tasks

## Done

- [x] Create ticket workspace
- [x] Draft detailed breaking-change implementation plan
- [x] Draft detailed execution task list

## Phase 1: Branch Domain Package

- [x] Create `pkg/wsm/branch/types.go` with `BranchName`, `RemoteName`, snapshot/request/plan types
- [x] Add enum types in `types.go`: `ResolutionMode`, `ResolutionStrategy`, `RemoteRefKind`
- [x] Create `pkg/wsm/branch/errors.go` with typed branch-domain errors
- [x] Create `pkg/wsm/branch/service.go` interface
- [x] Create `pkg/wsm/branch/resolver.go` deterministic strategy matrix
- [x] Add `pkg/wsm/branch/resolver_test.go` covering all strategy outcomes

## Phase 2: GitClient Primitive API Refactor (Breaking)

- [x] Replace ambiguous branch methods in `pkg/wsm/gitclient/client.go` with explicit local/remote-tracking primitives
- [x] Add explicit primitive methods: `ListLocalBranches`, `ListRemoteTrackingBranches`, `LocalBranchExists`, `RemoteTrackingBranchExists`
- [x] Remove/deprecate old ambiguous branch-policy method usage in interface and call sites
- [x] Implement new primitives in `pkg/wsm/gitclient/cli_client.go`
- [x] Implement new primitives in `pkg/wsm/gitclient/gogit_client.go`
- [x] Implement fallback-aware primitives in `pkg/wsm/gitclient/hybrid_client.go`
- [x] Add/extend backend contract tests for all new primitive methods

## Phase 3: Concrete BranchService

- [x] Implement `pkg/wsm/branch/service_impl.go`
- [x] Implement snapshot construction from backend primitives
- [x] Implement request validation + typed errors
- [x] Implement `Resolve` strategy generation per typed `ResolutionMode`
- [x] Ensure `BranchResolutionPlan` sets explicit `RemoteRefKind` for every strategy
- [x] Add `service_impl_test.go` with fixtures for local-only / remote-only / both / missing

## Phase 4: WorkspaceManager Migration

- [x] Inject `BranchService` into `WorkspaceManager` struct and constructor path
- [x] Remove duplicated branch state logic in `createWorktree` and `CreateWorktreeForAdd`
- [x] Replace branch decision code with `BranchService.Resolve`
- [x] Remove or rewrite `CheckBranchExists` and `CheckRemoteBranchExists` to new semantics
- [x] Ensure all branch-related errors are propagated (no ignored `_` branch-result errors)
- [x] Add focused unit tests for workspace behavior against branch plan outcomes

## Phase 5: Repo-Wide Caller Migration

- [x] Audit and migrate branch-related logic in `pkg/wsm/discovery.go`
- [x] Audit and migrate branch-related logic in `pkg/wsm/git_utils.go`
- [x] Audit and migrate branch-related logic in `pkg/wsm/sync_operations.go`
- [x] Audit and migrate branch-related logic in `pkg/wsm/rebase_operations.go` where applicable
- [x] Audit command-layer branch checks in `cmd/cmds/*` and migrate to new service
- [x] Run `rg -n "ListBranches|CheckBranchExists|CheckRemoteBranchExists|origin/"` and eliminate policy leakage

## Phase 6: Remove Legacy Paths

- [x] Delete obsolete branch helper functions superseded by BranchService
- [x] Remove comments/docs encoding old ambiguous branch semantics
- [x] Ensure no policy branching remains in backend adapters
- [x] Enforce compile-time breakage completion by removing compatibility shims

## Phase 7: Test and Validation

- [x] Run `go test ./pkg/wsm/branch/...` (new package)
- [x] Run `go test ./pkg/wsm/gitclient/...`
- [x] Run `go test ./pkg/wsm/...`
- [x] Run `go test ./...` and record non-ticket blockers separately
- [x] Add/update reproduction scripts under ticket `scripts/` for branch matrix validation
- [x] Capture before/after logs demonstrating removal of ambiguous branch behavior

## Phase 8: Documentation and Rollout

- [x] Update ticket design doc with implementation deltas and final API
- [x] Update `README.md` branch behavior notes
- [x] Update `IMPLEMENTATION.md` architecture section for new branch layer
- [x] Add changelog entry summarizing breaking branch API migration
- [x] Run `docmgr doctor --ticket WSM-MO-004-BRANCH-ABSTRACTION-REPO-WIDE --stale-after 30`
- [x] Close ticket after all tasks and validation complete
