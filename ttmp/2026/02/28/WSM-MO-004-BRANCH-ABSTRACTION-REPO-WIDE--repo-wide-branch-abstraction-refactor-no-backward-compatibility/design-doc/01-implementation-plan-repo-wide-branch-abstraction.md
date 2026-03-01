---
Title: 'Implementation Plan: Repo-Wide Branch Abstraction'
Ticket: WSM-MO-004-BRANCH-ABSTRACTION-REPO-WIDE
Status: active
Topics:
    - architecture
    - refactor
    - workspace-manager
    - git
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/client.go
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/cli_client.go
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/gogit_client.go
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/hybrid_client.go
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workspace.go
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/git_utils.go
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/discovery.go
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/git_integration.go
ExternalSources: []
Summary: "Breaking-change plan for introducing a repo-wide branch abstraction and removing legacy branch APIs/semantics."
LastUpdated: 2026-02-28T16:35:00-05:00
WhatFor: "Drive full repo branch-layer redesign on a clean baseline with no backwards compatibility constraints."
WhenToUse: "Use as the execution blueprint for the branch abstraction refactor ticket."
---

# Implementation Plan: Repo-Wide Branch Abstraction

## Executive Summary

This ticket defines a full repo-wide branch abstraction refactor with explicit **breaking changes**. The goal is to remove ambiguous branch semantics and consolidate all branch logic behind a single explicit branch domain model and service API.

Backward compatibility is explicitly out of scope. We will remove old method contracts and update all callers in one coherent migration.

## Implementation Delta (2026-02-28)

Implemented in code:

1. New branch domain package (`pkg/wsm/branch`) with typed enums:
   - `ResolutionMode`
   - `ResolutionStrategy`
   - `RemoteRefKind`
2. New branch policy service (`BranchService`) with deterministic resolver matrix.
3. Breaking `gitclient` API migration to explicit branch primitives:
   - `ListLocalBranches`
   - `ListRemoteTrackingBranches`
   - `LocalBranchExists`
   - `RemoteTrackingBranchExists`
4. Workspace flows (`createWorktree`, `CreateWorktreeForAdd`) migrated to `BranchService.Resolve`.
5. Repo-wide caller migration:
   - `sync_operations.go` branch switching now uses `BranchService.Resolve` with `ResolutionModeSync`
   - command-layer remote branch checks (`cmd_push.go`, `cmd_pr.go`, `cmd_rebase.go`) now use `BranchService`
   - `rebase_operations.go` and `git_utils.go` moved to typed remote-ref construction via `RemoteTrackingRef`.
6. Legacy wrappers removed from `WorkspaceManager`:
   - `CheckBranchExists`
   - `CheckRemoteBranchExists`
7. Added branch-centric tests:
   - resolver and service tests in `pkg/wsm/branch`
   - backend tests including go-git remote base ref creation
   - sync branch-switch matrix tests across cli/gogit/hybrid backends

Current non-ticket blocker:

- `go test ./...` still fails in integration scenarios due existing sandbox/discovery path issues (`open repo: repository does not exist`) unrelated to this branch abstraction migration.

## Problem Statement

Current branch behavior is fragmented and error-prone:

1. Different backends expose inconsistent branch semantics (local vs remote-tracking vs remote).
2. Callers in `pkg/wsm` previously inferred branch type from string formatting patterns.
3. Branch resolution logic was duplicated in workspace creation/add flows.
4. Branch operations are spread across `gitclient`, `workspace.go`, and `git_utils.go`.

Even after recent gap fixes, we still carry legacy API shape that allows misuse.

## Scope and Constraints

### In Scope

1. Introduce a new repo-wide branch abstraction layer.
2. Migrate all internal callers to it.
3. Delete or replace legacy branch APIs that preserve ambiguity.
4. Add comprehensive unit + workflow tests for new contract.

### Explicit Constraint

- **No backwards compatibility.**

### Out of Scope

1. UI/CLI ergonomics redesign beyond required wiring changes.
2. Rebase lifecycle redesign unrelated to branch state abstraction.

## Target Architecture

## New Branch Domain Model

Introduce typed branch concepts to prevent stringly-typed misuse.

```go
type BranchName string
type RemoteName string

type BranchSnapshot struct {
    LocalBranches          map[BranchName]struct{}
    RemoteTrackingBranches map[RemoteName]map[BranchName]struct{}
    CurrentBranch          BranchName
    Upstream               string // optional
}

type BranchResolutionRequest struct {
    TargetBranch BranchName
    BaseBranch   BranchName
    Remote       RemoteName
    Mode         ResolutionMode
}

type BranchResolutionPlan struct {
    LocalExists          bool
    RemoteTrackingExists bool
    Strategy             ResolutionStrategy
    StartPoint           string
    RemoteRefKind        RemoteRefKind
    RemoteRef            string
}

type ResolutionMode int
const (
    ResolutionModeUnspecified ResolutionMode = iota
    ResolutionModeCreateWorktree
    ResolutionModeAddRepository
    ResolutionModeSync
)

type ResolutionStrategy int
const (
    ResolutionStrategyUnspecified ResolutionStrategy = iota
    ResolutionStrategyUseLocal
    ResolutionStrategyTrackRemote
    ResolutionStrategyCreateFromBase
    ResolutionStrategyCreateFromHead
)

type RemoteRefKind int
const (
    RemoteRefKindNone RemoteRefKind = iota
    RemoteRefKindRemoteTrackingBranch
)
```

## New Service Boundary

```go
type BranchService interface {
    Snapshot(ctx context.Context, repo RepositoryHandle) (*BranchSnapshot, error)
    Resolve(ctx context.Context, repo RepositoryHandle, req BranchResolutionRequest) (*BranchResolutionPlan, error)

    LocalExists(ctx context.Context, repo RepositoryHandle, branch BranchName) (bool, error)
    RemoteTrackingExists(ctx context.Context, repo RepositoryHandle, remote RemoteName, branch BranchName) (bool, error)
    ListLocal(ctx context.Context, repo RepositoryHandle) ([]BranchName, error)
    ListRemoteTracking(ctx context.Context, repo RepositoryHandle, remote RemoteName) ([]BranchName, error)
}
```

## Backend Contract (gitclient)

`GitClient` will no longer be branch-policy-centric. It becomes a backend primitive layer consumed by `BranchService`.

### Breaking API Changes

1. Remove legacy ambiguous methods as branch decision source (for example, no caller branch policy based on `ListBranches`).
2. Keep only explicit branch primitives (local vs remote-tracking separated).
3. Require all backends to support explicit local and remote-tracking existence/listing methods.

## Design Decisions

1. **Typed branch domain objects and enums over free-form strings.**
2. **Single branch policy engine** (`BranchService.Resolve`) for all workflows.
3. **One-shot migration** with compile breaks accepted.
4. **No fallback to legacy behavior flags.**
5. **Deterministic test fixtures** for branch states (local-only, remote-only, both, missing).

## Breaking Changes (Intentional)

1. Existing callers that use old branch methods will fail compile until migrated.
2. Any semantics depending on legacy CLI normalization are removed.
3. Branch selection behavior becomes deterministic and explicit per strategy enum.

## File-Level Refactor Plan

## Phase 1: Introduce Branch Package

Create new package:

- `pkg/wsm/branch/`

Add:

1. `types.go` (domain structs/enums)
2. `service.go` (interface)
3. `resolver.go` (policy rules)
4. `errors.go` (typed branch errors)

## Phase 2: Expand Backend Primitives

Update:

1. `pkg/wsm/gitclient/client.go`
2. `pkg/wsm/gitclient/cli_client.go`
3. `pkg/wsm/gitclient/gogit_client.go`
4. `pkg/wsm/gitclient/hybrid_client.go`

Required primitive methods:

1. `LocalBranchExists`
2. `RemoteTrackingBranchExists` (rename current remote existence API to explicit tracking semantic)
3. `ListLocalBranches`
4. `ListRemoteTrackingBranches`

## Phase 3: Introduce Concrete BranchService

Create:

- `pkg/wsm/branch/service_impl.go`

Responsibilities:

1. Build snapshot from backend primitives.
2. Resolve branch strategy for each operation mode.
3. Produce a branch plan consumed by worktree/sync operations.
4. Use typed enums for mode and remote ref kind (`ResolutionMode`, `RemoteRefKind`).

## Phase 4: Migrate Workspace Flows

Update:

1. `pkg/wsm/workspace.go`
2. `pkg/wsm/git_integration.go`

Actions:

1. Inject `BranchService` into `WorkspaceManager`.
2. Replace duplicated branch decision code with `BranchService.Resolve`.
3. Remove old helper methods (`CheckBranchExists`, `CheckRemoteBranchExists`) or keep only as wrappers around new API during the same commit and delete before merge.

## Phase 5: Migrate Remaining Branch Call Sites

Update:

1. `pkg/wsm/discovery.go`
2. `pkg/wsm/git_utils.go`
3. `pkg/wsm/sync_operations.go`
4. any command-level branch checks in `cmd/cmds/*`

Actions:

1. Remove direct policy decisions from these files.
2. Use explicit branch queries from `BranchService`.

## Phase 6: Remove Legacy Branch Policy Paths

1. Delete stale branch decision helpers.
2. Remove legacy comments/docs describing old semantics.
3. Ensure no policy logic remains in backend adapters.

## Phase 7: Testing and Hardening

1. Add package-level tests for `branch/resolver.go` covering all decision matrix cases.
2. Add backend contract tests for branch primitive methods.
3. Add workspace integration-style unit tests with deterministic fixtures.
4. Ensure no branch policy in code paths bypasses `BranchService`.

## Phase 8: Documentation and Finalization

1. Update `IMPLEMENTATION.md` and `README.md` branch architecture sections.
2. Add migration notes describing intentional breaking changes.
3. Link follow-on tickets (if needed) for rebase/sync-specific policy refinements.

## Decision Matrix for Branch Resolution

Define deterministic behavior for `BranchResolutionRequest.Mode` and strategy outcomes:

1. Local exists => `use-local`
2. Local missing + remote-tracking exists => `track-remote`
3. Local missing + remote-tracking missing + base provided => `create-from-base`
4. Local missing + remote-tracking missing + no base => `create-from-head`

No implicit fallback to ambiguous lists.
`RemoteRefKind` must always be explicit in `BranchResolutionPlan` (`None` or `RemoteTrackingBranch`).

## Testing Strategy

## Unit Tests

1. Resolver matrix tests for all strategy outcomes.
2. Typed error tests (invalid request, empty branch name, unknown mode).
3. Backend primitive tests for CLI/go-git/hybrid parity.

## Workflow Tests

1. `create workspace` with existing local branch.
2. `create workspace` with remote-only tracking branch.
3. `add repo` with base branch fallback.
4. missing branch with no base.

## Regression Tests

1. Ensure previous false-negative gap does not recur.
2. Ensure no silent branch source ambiguity in CLI mode.

## Risks and Mitigations

1. Risk: large compile-breaking diff.
   - Mitigation: phase commits by package boundary with green tests between phases.
2. Risk: hidden callers still rely on old semantics.
   - Mitigation: ripgrep sweep + compile break intentionally enforced.
3. Risk: integration suite noise from unrelated harness issues.
   - Mitigation: prioritize deterministic unit/workflow tests in package scope; track harness separately.

## Alternatives Considered

1. Incremental compatibility shims.
   - Rejected: prolongs ambiguous semantics.
2. Keep branch policy in `WorkspaceManager` only.
   - Rejected: duplicates policy in future call sites.
3. Backend-specific policy branches.
   - Rejected: undermines consistent behavior target.

## Completion Criteria

1. All branch decisions flow through `BranchService.Resolve`.
2. No caller uses ambiguous branch lists for policy.
3. Legacy branch policy helpers removed.
4. New branch package + tests merged.
5. Ticket tasks closed and docs updated.

## References

1. `pkg/wsm/gitclient/client.go`
2. `pkg/wsm/gitclient/cli_client.go`
3. `pkg/wsm/gitclient/gogit_client.go`
4. `pkg/wsm/gitclient/hybrid_client.go`
5. `pkg/wsm/workspace.go`
6. `pkg/wsm/git_utils.go`
7. `pkg/wsm/discovery.go`
8. `pkg/wsm/git_integration.go`
