---
Title: 'Bug Report and Fix Plan: Remote Branch Detection'
Ticket: WSM-MO-002-REF-GAP1-REMOTE-BRANCH
Status: active
Topics:
    - refactor
    - architecture
    - workspace-manager
    - git
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/wsm/gitclient/cli_client.go
      Note: CLI branch normalization behavior causing mismatch
    - Path: pkg/wsm/gitclient/client.go
      Note: GitClient contract extension target
    - Path: pkg/wsm/gitclient/gogit_client.go
      Note: go-git ListBranches returns local refs only
    - Path: pkg/wsm/workspace.go
      Note: Caller paths and faulty CheckRemoteBranchExists implementation
    - Path: ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.log
      Note: Captured failing run evidence
    - Path: ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.sh
      Note: Deterministic bug reproduction script
ExternalSources: []
Summary: Detailed bug report and implementation plan for false-negative remote branch detection during worktree creation flows.
LastUpdated: 2026-02-28T14:20:00-05:00
WhatFor: Hand-off quality bug analysis and fix plan for intern onboarding.
WhenToUse: Use when implementing or reviewing remote-branch existence logic in workspace creation/add flows.
---


# Bug Report and Fix Plan: Remote Branch Detection

## Executive Summary

This ticket documents a correctness bug in refactor gap 1: `CheckRemoteBranchExists` can return `false` even when `origin/<branch>` exists. The defect is deterministic under `cli`, `gogit`, and `hybrid` backends and is reproduced by an executable script in this ticket.

The immediate cause is an API-contract mismatch:

1. `CheckRemoteBranchExists` assumes `ListBranches` returns values prefixed with `origin/`.
2. `CliGitClient.ListBranches` strips `remotes/origin/` and returns short names.
3. `GoGitClient.ListBranches` returns local branches only.

Because `createWorktree` and `CreateWorktreeForAdd` rely on `CheckRemoteBranchExists`, branch resolution can choose the wrong creation path, producing local branches from `HEAD` instead of correctly basing on remote branch state.

## Bug Report

### ID and Severity

- Ticket ID: `WSM-MO-002-REF-GAP1-REMOTE-BRANCH`
- Severity: `High`
- Category: correctness + architecture contract mismatch
- Affected area: workspace creation and add-repo worktree flow

### Impacted User Flows

1. `CreateWorkspace` path via `createWorktree`.
2. `AddRepositoryToWorkspace` path via `CreateWorktreeForAdd`.

### Evidence Anchors

1. `createWorktree` invokes `CheckRemoteBranchExists` and branches behavior based on result:
   - `pkg/wsm/workspace.go:250-302`
2. `CreateWorktreeForAdd` repeats same decision pattern:
   - `pkg/wsm/workspace.go:1029-1089`
3. `CheckRemoteBranchExists` implementation checks `ListBranches` output for `origin/` prefix:
   - `pkg/wsm/workspace.go:321-334`
4. CLI branch listing strips remote prefix:
   - `pkg/wsm/gitclient/cli_client.go:50-62`
5. go-git branch listing iterates local branches only:
   - `pkg/wsm/gitclient/gogit_client.go:49-59`

### Reproduction Evidence

Script:

- `scripts/repro_remote_branch_false_negative.sh`

Observed output (`scripts/repro_remote_branch_false_negative.log`):

1. Remote branch is present by ground-truth command:
   - `git ls-remote --heads origin feature/remote-only` returns `refs/heads/feature/remote-only`.
2. Actual method result:
   - `backend=cli exists=false err=<nil>`
   - `backend=gogit exists=false err=<nil>`
   - `backend=hybrid exists=false err=<nil>`

This proves the method false-negative in all configured backends.

## Why This Happens (Root Cause)

### Root Cause 1: Semantic Mismatch Between API and Caller Expectations

`CheckRemoteBranchExists` treats `ListBranches` as if it returned remote-qualified refs like `origin/foo`. That assumption is invalid for both current implementations.

1. CLI backend normalizes `remotes/origin/foo` to `foo`.
2. go-git backend provides only local branch refs.

So the condition:

```go
if strings.HasPrefix(b, "origin/") && strings.TrimPrefix(b, "origin/") == branch
```

is almost never true.

### Root Cause 2: Error Handling Is Ignored in Several Steps

In both worktree decision paths, errors from `gc.ListBranches` and `CheckRemoteBranchExists` are ignored (`branches, _ := ...`, `remoteBranchExists, _ := ...`). This can silently force fallback behavior even when backend calls fail.

### Root Cause 3: Duplicated Branch-Resolution Logic

Branch existence checks are duplicated in two call paths (`createWorktree`, `CreateWorktreeForAdd`) with near-identical logic. Duplication increases maintenance risk and encourages drift.

## Concepts the Intern Must Understand

### 1) Local Branch vs Remote-Tracking Branch vs Remote Branch

- Local branch: `refs/heads/<name>`.
- Remote-tracking branch: `refs/remotes/origin/<name>`.
- Remote branch (server state): queried via network (for example `git ls-remote`).

These are related but not identical states.

### 2) `git worktree add` Branch Semantics

In this codebase, `CliWorktrees.Add` builds commands like:

- new branch: `git worktree add -b <branch> <target> [<start-point>]`
- overwrite mode: `git worktree add -B <branch> ...`

If we fail to detect remote branch existence, we may create from wrong start-point.

### 3) Contract Design Matters More Than Convenience

`ListBranches` is introspection. `RemoteBranchExists` is a specific business question. Reusing an approximate API for a precise question caused this bug.

## Scope

### In Scope for This Ticket

1. Reliable remote branch existence detection for `origin` by default.
2. Removal of false-negative behavior in worktree branch selection.
3. Error propagation in branch-resolution path (no silent `_` ignore where it controls behavior).
4. Test coverage for `cli`, `gogit`, and `hybrid` behavior boundaries.

### Out of Scope

1. Full refactor of all branch logic across unrelated features.
2. New remote management UX.
3. Large redesign of all GitClient methods unrelated to this bug.

## Proposed Fix Design

## Decision

Introduce a dedicated GitClient capability for remote-branch existence and migrate callers to it.

### API Change

Add method to `GitClient` interface in `pkg/wsm/gitclient/client.go`:

```go
RemoteBranchExists(ctx context.Context, repo RepositoryHandle, remote string, branch string) (bool, error)
```

Rationale:

1. Makes intent explicit.
2. Avoids brittle parsing assumptions from `ListBranches`.
3. Enables backend-specific correct implementation.

### Backend Implementations

#### CLI Backend (`cli_client.go`)

Primary check (local tracking refs, no network):

```bash
git show-ref --verify --quiet refs/remotes/<remote>/<branch>
```

Fallback check (network truth, if desired or if local ref missing and we opt in):

```bash
git ls-remote --exit-code --heads <remote> refs/heads/<branch>
```

Implementation detail decision:

1. Default to local-tracking check for speed and offline behavior.
2. Optional enhancement: perform network check when configured by env flag (future).

#### go-git Backend (`gogit_client.go`)

Check ref existence for:

- `refs/remotes/<remote>/<branch>`

If missing, return `false,nil`.

#### Hybrid Backend (`hybrid_client.go`)

Use standard fallback semantics:

1. Try primary.
2. If `errors.Is(err, ErrNotImplemented)`, use fallback.
3. Otherwise return original error.

### WorkspaceManager Changes (`workspace.go`)

1. Update `CheckRemoteBranchExists` to use new method directly.
2. Stop scanning `ListBranches` for `origin/` prefixes.
3. Stop suppressing critical errors during branch-resolution decisions.

Suggested behavior:

1. If branch check errors, return wrapped error and abort creation path.
2. Keep business decision deterministic and auditable.

### De-duplication Change

Extract shared logic for branch resolution into helper, e.g.:

```go
type branchResolution struct {
    localExists bool
    remoteExists bool
}

func (wm *WorkspaceManager) resolveBranchState(...) (branchResolution, error)
```

Used by both `createWorktree` and `CreateWorktreeForAdd`.

This removes copy-paste and keeps one policy.

## Pseudocode

```go
func (wm *WorkspaceManager) CheckRemoteBranchExists(ctx context.Context, repoPath, branch string) (bool, error) {
    gc, _ := BuildGitBackends(ctx)
    h, err := gc.Open(ctx, repoPath)
    if err != nil {
        return false, errors.Wrap(err, "open repository")
    }

    exists, err := gc.RemoteBranchExists(ctx, h, "origin", branch)
    if err != nil {
        return false, errors.Wrap(err, "check remote branch existence")
    }
    return exists, nil
}
```

```go
func (c *CliGitClient) RemoteBranchExists(ctx context.Context, repo RepositoryHandle, remote, branch string) (bool, error) {
    if remote == "" { remote = "origin" }
    ref := fmt.Sprintf("refs/remotes/%s/%s", remote, branch)
    _, err := runGit(ctx, repo.Path(), "show-ref", "--verify", "--quiet", ref)
    if err == nil {
        return true, nil
    }
    if isExitCodeOne(err) { // not found
        return false, nil
    }
    return false, err
}
```

## Alternatives Considered

### Alternative A: Keep Existing API and Adjust Parsing

Idea: keep `ListBranches`, but make all backends include remote prefixes.

Rejected because:

1. Conflates local and remote semantics in one list.
2. Creates ambiguity for other callers expecting local branches.
3. Increases formatting coupling and brittleness.

### Alternative B: Use `git ls-remote` Everywhere

Idea: always network query remote branch existence.

Rejected as default because:

1. Adds network dependency and latency to local operation.
2. Breaks offline workflows.
3. More failure modes (auth/network) in core flow.

### Alternative C: Keep False-Negative-Tolerant Behavior

Rejected due correctness risk and hidden branch base mistakes.

## Detailed Implementation Plan

### Phase 1: API and Backends

1. Add `RemoteBranchExists` to `GitClient` interface.
2. Implement in `CliGitClient`.
3. Implement in `GoGitClient`.
4. Implement in `HybridClient` with strict fallback behavior.

### Phase 2: Callers

1. Rewrite `WorkspaceManager.CheckRemoteBranchExists` to call the dedicated method.
2. Replace `_`-ignored errors in branch-selection path with explicit handling.
3. Introduce shared helper for branch-resolution to remove duplication.

### Phase 3: Tests

1. Add backend-focused unit tests in `pkg/wsm/gitclient/*_test.go`.
2. Add workspace behavior tests in `pkg/wsm/workspace*_test.go` or integration scenario.
3. Add regression case proving remote branch is detected when only remote-tracking ref exists.

### Phase 4: Validation

1. Run unit tests.
2. Run targeted integration scenario for worktree creation/add.
3. Re-run `scripts/repro_remote_branch_false_negative.sh`; expected outcome after fix:
   - `exists=true` for at least `cli` and `hybrid` with fetched tracking ref.

## Test Design (Concrete)

### Unit Tests

1. `CliGitClient.RemoteBranchExists`:
   - found ref returns `true,nil`
   - missing ref returns `false,nil`
   - git command failure returns error
2. `GoGitClient.RemoteBranchExists`:
   - same matrix using test repo fixtures
3. `HybridClient.RemoteBranchExists`:
   - primary success => fallback not called
   - primary not-implemented => fallback called
   - primary real error => returned unchanged

### Workflow Tests

1. `createWorktree` chooses `opts.RemoteBranch` when remote exists.
2. `CreateWorktreeForAdd` chooses `opts.RemoteBranch` when remote exists.
3. error during remote check aborts operation with explicit error.

## Risks and Mitigations

1. Risk: interface expansion breaks compile for all implementations.
   - Mitigation: update all backends in one commit.
2. Risk: behavior changes in edge repos with stale refs.
   - Mitigation: clearly document local-tracking behavior and optionally add explicit fetch policy later.
3. Risk: larger diff due helper extraction.
   - Mitigation: phase changes; keep helper limited to branch existence state only.

## Acceptance Criteria

1. Reproduction script no longer prints false for valid remote branch in intended backend path.
2. No ignored errors in branch-resolution decisions controlling worktree add mode.
3. Tests cover both positive and negative remote existence paths.
4. Code duplication around branch resolution reduced to one helper.

## Intern Runbook

1. Read this design doc front-to-back.
2. Run reproduction script and inspect log before changing code.
3. Implement interface and backend methods first.
4. Patch workspace call sites and remove ignored errors.
5. Add tests next.
6. Re-run script and tests; attach outputs to PR.

## References

1. `pkg/wsm/workspace.go:250-334`
2. `pkg/wsm/workspace.go:1029-1089`
3. `pkg/wsm/gitclient/cli_client.go:50-62`
4. `pkg/wsm/gitclient/gogit_client.go:49-59`
5. `pkg/wsm/gitclient/client.go:47-74`
6. `pkg/wsm/gitclient/worktree_cli.go:14-30`
7. `pkg/wsm/git_integration.go:10-23`
8. `scripts/repro_remote_branch_false_negative.sh`
9. `scripts/repro_remote_branch_false_negative.log`
