---
Title: Refactor Status and Target Architecture
Ticket: WSM-MO-001-ANALYZE-REFACTOR
Status: active
Topics:
    - refactor
    - architecture
    - workspace-manager
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: workspace-manager/pkg/wsm/git_integration.go
      Note: Backend selection via WSM_GIT_BACKEND
    - Path: workspace-manager/pkg/wsm/gitclient/client.go
      Note: GitClient and WorktreeManager interfaces
    - Path: workspace-manager/pkg/wsm/gitclient/gogit_client.go
      Note: Go-git backend with unimplemented methods
    - Path: workspace-manager/pkg/wsm/rebase_operations.go
      Note: CLI-only rebase flow
    - Path: workspace-manager/pkg/wsm/status.go
      Note: Status checker using GitClient
    - Path: workspace-manager/pkg/wsm/sync_operations.go
      Note: Sync operations mixing GitClient and CLI
    - Path: workspace-manager/pkg/wsm/workspace.go
      Note: Core workspace orchestration; mixed git backends
ExternalSources: []
Summary: Detailed analysis of the current refactor state and the desired target architecture for workspace-manager.
LastUpdated: 2026-02-28T13:38:15-05:00
WhatFor: Understand current vs desired architecture and the remaining refactor work.
WhenToUse: Use when continuing the refactor or onboarding new contributors.
---


# Refactor Status and Target Architecture

## Purpose and Audience

This document captures the current architecture of workspace-manager (WSM), the state of the recent refactor, and the desired end-state architecture. It is written for maintainers and contributors who need to understand how the system is structured today and what remains to complete the refactor.

## Scope

- Focus: core architecture and refactor status in `workspace-manager/`.
- Emphasis: Git abstraction refactor (CLI vs go-git backends), command/core separation, and operational flows.
- Out of scope: new features, behavior changes, or performance tuning.
- Validation pass: claims re-checked against repository state on **2026-02-28**.

## Glossary

- **Workspace**: A multi-repository worktree directory (one worktree per repo) with metadata stored in `~/.config/workspace-manager/workspaces/`.
- **Repository**: A discovered git repository tracked in a registry (`registry.json`).
- **Worktree**: A git worktree created under the workspace path.
- **Backend**: The git implementation (go-git, CLI, or hybrid) used for git operations.

## Current Architecture (As Implemented)

### Repository Layout

- `cmd/wsm/`: binary entry point and root Cobra command wiring.
- `cmd/cmds/`: per-command implementations that call into `pkg/wsm`.
- `pkg/wsm/`: core workspace logic (create, status, sync, delete, etc.).
- `pkg/wsm/gitclient/`: Git abstraction layer with go-git and CLI backends.
- `pkg/output/`: user-facing output formatting.

### Current Component Diagram

```
+-------------------+         +--------------------+
| cmd/wsm (main)    |         | cmd/cmds/*          |
| - Cobra root      |  calls  | - command handlers |
+-------------------+-------->+--------------------+
                                       |
                                       v
                              +--------------------+
                              | pkg/wsm            |
                              | - WorkspaceManager |
                              | - StatusChecker    |
                              | - GitOperations    |
                              | - SyncOperations   |
                              | - Rebase helpers   |
                              +--------------------+
                                       |
                                       v
                              +--------------------+
                              | pkg/wsm/gitclient  |
                              | - GitClient        |
                              | - WorktreeManager  |
                              +--------------------+
                                 /            \
                                v              v
                          +---------+     +---------+
                          | go-git  |     | git CLI |
                          +---------+     +---------+
```

### Core Data Models (Observed)

From `pkg/wsm/types.go`:

```go
// Workspace represents a multi-repository workspace
// (simplified)
type Workspace struct {
    Name         string
    Path         string
    Repositories []Repository
    Branch       string
    BaseBranch   string
    Created      time.Time
    GoWorkspace  bool
    AgentMD      string
}
```

```go
// Repository represents a discovered git repository
// (simplified)
type Repository struct {
    Name          string
    Path          string
    RemoteURL     string
    CurrentBranch string
    Branches      []string
    Tags          []string
    LastCommit    string
    Categories    []string
}
```

### Git Abstraction Layer (Current)

From `pkg/wsm/gitclient/client.go`:

```go
// GitClient defines repository-level git operations.
type GitClient interface {
    Open(ctx context.Context, repoPath string) (RepositoryHandle, error)
    CurrentBranch(ctx context.Context, repo RepositoryHandle) (string, error)
    RemoteURL(ctx context.Context, repo RepositoryHandle, remote string) (string, error)
    ListBranches(ctx context.Context, repo RepositoryHandle) ([]string, error)
    ListTags(ctx context.Context, repo RepositoryHandle) ([]string, error)
    LastCommit(ctx context.Context, repo RepositoryHandle) (string, error)
    Status(ctx context.Context, repo RepositoryHandle) (Status, error)
    Add(ctx context.Context, repo RepositoryHandle, path string) error
    Reset(ctx context.Context, repo RepositoryHandle, path string) error
    Commit(ctx context.Context, repo RepositoryHandle, msg string, opts CommitOptions) (string, error)
    Diff(ctx context.Context, repo RepositoryHandle, staged bool) (string, error)
    Fetch(ctx context.Context, repo RepositoryHandle, remote string) error
    Push(ctx context.Context, repo RepositoryHandle, remote string) error
    AheadBehind(ctx context.Context, repo RepositoryHandle, upstream string) (ahead int, behind int, err error)
    CreateBranch(ctx context.Context, repo RepositoryHandle, name string, track bool, baseRef string) error
    CheckoutBranch(ctx context.Context, repo RepositoryHandle, name string, create bool, force bool) error
}

// WorktreeManager defines operations for git worktrees.
type WorktreeManager interface {
    Add(ctx context.Context, repoPath string, branch string, targetPath string, opts WorktreeAddOptions) error
    Remove(ctx context.Context, repoPath string, targetPath string, force bool) error
    List(ctx context.Context, repoPath string) ([]WorktreeInfo, error)
}
```

Backend selection lives in `pkg/wsm/git_integration.go`:

- Env `WSM_GIT_BACKEND` chooses `hybrid` (default), `gogit`, or `cli`.
- Worktrees are always CLI (via `WorktreeManager`).

### Operational Flows (Current)

#### Workspace Creation (Observed)

```text
CreateWorkspace(ctx, name, repoNames, branch, baseBranch, agentSource, dryRun):
  if name empty -> error
  repos = FindRepositories(repoNames)
  workspace = {Name, Path, Branch, BaseBranch, Repositories, Created, GoWorkspace, AgentMD}
  if dryRun -> return workspace
  createWorkspaceStructure:
    mkdir workspace.Path
    for each repo in workspace.Repositories:
      createWorktree(repo)
      if create fails -> rollback worktrees + cleanup dir
    if workspace.GoWorkspace -> CreateGoWorkspace
    if workspace.AgentMD -> copyAgentMD
    run setup scripts (best-effort)
  SaveWorkspace(workspace)
  return workspace
```

Notes:
- Worktree creation uses `WorktreeManager` but branch checks mix GitClient + custom logic.
- Rollback and some verification paths still call `git` directly.

#### Status (Observed)

```text
StatusChecker.GetWorkspaceStatusWithOptions:
  gc = BuildGitBackends()
  for each repo (concurrent if jobs > 1):
    handle = gc.Open(repoPath)
    st = gc.Status(handle)
    ahead/behind = gc.AheadBehind(handle)
    populate RepositoryStatus
  return WorkspaceStatus
```

#### Sync (Observed)

```text
SyncOperations.SyncWorkspace:
  for each repo (concurrent if jobs > 1):
    ahead/behind = gc.AheadBehind
    if dry-run -> stop
    if pull -> gc.Fetch + optional CLI rebase
    if push -> gc.Push
    ahead/behind again
```

#### Rebase (Observed)

Rebase operations in `pkg/wsm/rebase_operations.go` are CLI-first and use porcelain parsing for conflict detection.

## Refactor Status (Where We Are Now)

### What Has Been Migrated to GitClient

- **Status**: `pkg/wsm/status.go` uses `GitClient` for status and ahead/behind.
- **Discovery**: `pkg/wsm/discovery.go` prefers `GitClient` and falls back to CLI helpers.
- **Commit/Stage/Status**: `pkg/wsm/git_operations.go` uses `GitClient` for status, add, commit, push.
- **Worktree Creation**: `pkg/wsm/workspace.go` uses `WorktreeManager.Add` for creation.

### What Still Uses Direct CLI Calls

From `rg "exec.CommandContext" pkg/wsm`:

- `pkg/wsm/workspace.go`: worktree debug listing (`git worktree list`), rollback removal (`git worktree remove --force`), untracked file checks, setup script execution.
- `pkg/wsm/discovery.go`: CLI fallbacks for remote URL/branch/tags.
- `pkg/wsm/git_utils.go`: branch merged/rebase checks (fetch + merge-base + rev-list).
- `pkg/wsm/rebase_operations.go`: full rebase workflow and conflict detection.
- `pkg/wsm/sync_operations.go`: conflict detection and log retrieval.

### Refactor Gaps and Friction

1. **Remote branch detection is currently inconsistent (and can fail in all backends)**
   - `CheckRemoteBranchExists` uses `ListBranches` and expects `origin/<branch>` entries.
   - go-git `ListBranches` returns local branch refs only; it does not include remote refs.
   - CLI `ListBranches` normalizes by trimming `remotes/origin/`, so returned values are not `origin/<branch>` either.
   - Net effect: `CheckRemoteBranchExists` can incorrectly return false even when `origin/<branch>` exists.

2. **HybridClient drops non-fallback errors in several methods**
   - In `gitclient/hybrid_client.go`, methods like `Add`, `Reset`, `Fetch`, `Push`, `CreateBranch`, and `CheckoutBranch` return `nil` unless the error is exactly `ErrNotImplemented`.
   - This means real backend failures from the primary client can be silently swallowed in hybrid mode.
   - This is a correctness risk and should be fixed before deeper refactor sequencing.

3. **go-git backend is incomplete**
   - `Reset`, `Diff`, and `AheadBehind` are not implemented in `pkg/wsm/gitclient/gogit_client.go`.
   - In hybrid mode, missing operations fall back to CLI when `ErrNotImplemented` is returned.
   - In pure `gogit` mode, callers may get degraded behavior (e.g., no ahead/behind).

4. **Backend construction is duplicated across hot paths**
   - `BuildGitBackends` is called repeatedly in `workspace.go`, `status.go`, `sync_operations.go`, `git_operations.go`, and `discovery.go`.
   - Backend lifecycle and configuration are not injected once into long-lived services (`WorkspaceManager`, `StatusChecker`, `SyncOperations`).
   - This increases coupling and makes behavior harder to reason about/test.

5. **Worktree lifecycle remains partially split**
   - Positive progress: `removeWorktrees` and `removeWorktreeForRepo` now use `WorktreeManager.Remove` for actual removal.
   - Remaining split: rollback (`rollbackWorktrees`) and debug listing still shell out to `git worktree`.
   - Result: behavior is improved but still not fully centralized under one abstraction.

6. **Mixed responsibility across layers**
   - Some git operations are in `pkg/wsm/git_utils.go`, others in `pkg/wsm/gitclient`, and others are ad-hoc in `workspace.go` and `sync_operations.go`.
   - The abstraction boundary is not yet clean, which complicates testing and backend selection.

7. **Documentation drift**
   - `README.md` still references `go build ./cmd/workspace-manager` (at least two places), while install docs use `cmd/wsm`.

### Double-Check Corrections Applied (2026-02-28)

- Corrected prior statement that delete/worktree removal is fully CLI-bound: removal now calls `WorktreeManager.Remove`, while rollback/listing still use direct CLI.
- Expanded remote-branch detection finding to include the CLI normalization mismatch (not only go-git behavior).
- Added a newly identified high-risk gap: `HybridClient` swallowing non-`ErrNotImplemented` errors in several mutation/sync methods.
- Confirmed README drift still exists for build path examples.

### Migration Map (Snapshot)

| Area | Current Implementation | Backend/Notes |
| --- | --- | --- |
| Status | `pkg/wsm/status.go` | GitClient (hybrid/default) |
| Discovery | `pkg/wsm/discovery.go` | GitClient + CLI fallbacks |
| Create workspace | `pkg/wsm/workspace.go` | WorktreeManager for add; CLI for rollback/listing/untracked/setup scripts |
| Delete workspace | `pkg/wsm/workspace.go` | WorktreeManager for remove; CLI for listing/untracked checks and rollback path |
| Commit | `pkg/wsm/git_operations.go` | GitClient |
| Sync | `pkg/wsm/sync_operations.go` | GitClient for fetch/push; CLI for rebase/conflicts |
| Rebase | `pkg/wsm/rebase_operations.go` | CLI only |
| Branch merge checks | `pkg/wsm/git_utils.go` | CLI only |

## Desired Architecture (Target End State)

### Goals

- **Single Git abstraction boundary**: all git operations go through `GitClient` or an explicit `GitService` adapter; no direct `exec` calls in `pkg/wsm` except inside backend implementations.
- **Predictable backend selection**: `BuildGitBackends` invoked once, injected into core services.
- **Testability**: ability to swap a fake backend to unit test flows without shelling out.
- **Behavioral consistency**: uniform status/branch detection semantics across backends.

### Target Component Diagram

```
+-------------------+         +--------------------+
| cmd/wsm (main)    |         | cmd/cmds/*          |
+-------------------+-------->+--------------------+
                                       |
                                       v
                              +--------------------+
                              | pkg/wsm            |
                              | - WorkspaceManager |
                              | - StatusChecker    |
                              | - GitOperations    |
                              | - SyncOperations   |
                              +--------------------+
                                       |
                                       v
                              +--------------------+
                              | GitService         |
                              | - GitClient        |
                              | - WorktreeManager  |
                              | - Rebase API       |
                              +--------------------+
                                       |
                                       v
                          +-----------------------------+
                          | gitclient backends          |
                          | - go-git (primary)          |
                          | - cli (fallback)            |
                          +-----------------------------+
```

### Target API Boundaries (Illustrative)

```go
// GitService centralizes all git operations needed by pkg/wsm.
type GitService interface {
    Client() gitclient.GitClient
    Worktrees() gitclient.WorktreeManager
    Rebase() RebaseManager
    Branches() BranchManager
}

// RebaseManager abstracts rebase flows (could remain CLI-only internally).
type RebaseManager interface {
    Start(ctx context.Context, repoPath, upstream string, opts RebaseOptions) error
    Continue(ctx context.Context, repoPath string) error
    Abort(ctx context.Context, repoPath string) error
    Status(ctx context.Context, repoPath string) (RebaseState, []ConflictInfo, error)
}
```

### Target Flow (Example: Worktree Removal)

```text
DeleteWorkspace(ctx, name, removeFiles, forceWorktrees):
  workspace = LoadWorkspace(name)
  for each repo:
    worktreePath = workspace.Path/repo.Name
    untracked = GitService.Client().Status(repo)
    if untracked and !force -> error
    GitService.Worktrees().Remove(repo.Path, worktreePath, force)
  cleanup workspace dir
  delete workspace config
```

### Migration Priorities (Recommended)

1. **Fix HybridClient error propagation first**: ensure mutation/sync methods return non-`ErrNotImplemented` errors instead of swallowing them.
2. **Centralize backend selection**: construct GitService once (e.g., in `NewWorkspaceManager`) and pass to helpers.
3. **Eliminate direct `exec` in core**: move remaining CLI usage into `gitclient` or explicit adapters.
4. **Improve backend semantics**:
   - Add remote-branch listing to GoGit backend (or a new API for remote refs).
   - Implement `AheadBehind` and `Diff` in go-git or keep explicit fallback logic.
5. **Normalize conflict detection**: reuse GitClient status output or add a dedicated conflict-check API.

### Migration Priority Options (Detailed)

The priority choice in the previous section can be executed several ways. Below are concrete options, each emphasizing a different risk profile and payoff curve. These options are not mutually exclusive; they are sequencing strategies.

#### Option A: Coverage-First (GitService boundary + CLI containment)

**Intent:** stop `pkg/wsm` from shelling out directly as soon as possible, even if go-git remains incomplete. This yields a clean boundary that can be tested and evolved later.

**Steps (incremental):**
1. Introduce `GitService` and construct it once in `NewWorkspaceManager`.
2. Pass `GitService` into `StatusChecker`, `GitOperations`, `SyncOperations`, and helpers.
3. Move remaining CLI calls (rebase, worktree list/remove, conflict checks) behind `GitService` adapters.
4. Keep hybrid fallback behavior for missing go-git operations.

**Pros:**
- Clean architecture boundary quickly.
- Enables deterministic test doubles for git operations.
- Reduces implicit behavior hidden in ad-hoc exec calls.

**Cons:**
- Still depends on CLI for incomplete go-git operations.
- Hybrid behavior may obscure which backend is active.

**Where it lands in the code:**
- `workspace-manager/pkg/wsm/workspace.go`: replace direct `exec` calls with `GitService.Worktrees()` and `GitService.Rebase()`.
- `workspace-manager/pkg/wsm/git_utils.go`: merge into `GitService.Branches()` or `GitService.Client()` helpers.

**Sketch:**
```go
type WorkspaceManager struct {
    config     *WorkspaceConfig
    Discoverer *RepositoryDiscoverer
    git        GitService
}

func NewWorkspaceManager() (*WorkspaceManager, error) {
    // ...
    git := NewGitServiceFromEnv()
    return &WorkspaceManager{config: config, Discoverer: discoverer, git: git}, nil
}
```

#### Option B: Backend Parity-First (go-git completeness)

**Intent:** reduce CLI reliance by filling go-git gaps before reorganizing the core. This makes the backend choice meaningful and improves predictability in `gogit` mode.

**Primary gaps to close:**
- `AheadBehind` in `pkg/wsm/gitclient/gogit_client.go`.
- `Diff` implementation (or a structured diff API instead of unified diffs).
- `Reset` with path semantics (or explicit CLI fallback inside go-git backend).
- Remote branch visibility (currently only local branches are returned).

**Pros:**
- Backend behavior is more uniform regardless of selection.
- Simplifies future logic (less fallback complexity).

**Cons:**
- Higher implementation complexity (go-git does not expose all CLI semantics).
- Delays architectural cleanup in `pkg/wsm`.

**Remote refs approach (design option):**
```go
type GitClient interface {
    ListBranches(ctx context.Context, repo RepositoryHandle) ([]string, error)
    ListRemoteBranches(ctx context.Context, repo RepositoryHandle) ([]string, error)
}
```
This avoids overloading `ListBranches` and removes ambiguity between local and remote results.

#### Option C: Worktree and Lifecycle Consistency First

**Intent:** stabilize workspace creation/deletion flows (the highest-risk operations) before deeper git refactors.

**Focus areas:**
- `CreateWorkspace`, `DeleteWorkspace`, rollback behavior.
- Worktree existence checks and untracked file gating.
- Consistent error handling and rollback semantics.

**Pros:**
- Reduces operational risk (data loss, orphaned worktrees).
- Improves user trust in destructive commands.

**Cons:**
- Does not address broader backend design.

**Lifecycle flow emphasis:**
```
Create -> WorktreeAdd -> GoWork -> AgentMD -> SetupScripts -> Save
   \-> On failure: WorktreeRemove (reverse order) -> CleanupDir
Delete -> WorktreeRemove -> CleanupWorkspace -> RemoveConfig
```

#### Option D: Behavior Contracts and Tests First

**Intent:** lock down existing behavior before refactoring so regressions are visible.

**Test targets:**
- Repository discovery and registry updates.
- Workspace creation/deletion (dry-run + actual).
- Status aggregation and ahead/behind semantics.
- Rebase conflict detection and recovery.

**Pros:**
- Safe refactor baseline; encourages confidence in later changes.
- Forces clarity on ambiguous behavior.

**Cons:**
- Adds upfront time before architecture changes.

#### Decision Matrix (Guidance)

```
Option  | Speed | Risk | Architecture Cleanliness | Backend Consistency
--------|-------|------|--------------------------|--------------------
A       | High  | Low  | High                     | Medium
B       | Low   | Med  | Medium                   | High
C       | Med   | Low  | Medium                   | Medium
D       | Low   | Low  | Medium                   | Medium
```

#### Possible Sequencing Recipes

1. **A -> C -> B**
   - Clean boundary quickly, stabilize worktree lifecycle, then improve go-git parity.
2. **D -> A -> B**
   - Lock behavior, then consolidate, then fill backend gaps.
3. **C -> A**
   - If users are seeing worktree cleanup issues, stabilize lifecycle first, then centralize.
4. **Immediate pre-step**
   - Before any sequence, patch `HybridClient` error handling to avoid false-success operations in hybrid mode.

#### What “done” looks like

- All core flows in `pkg/wsm` use `GitService` APIs; no direct `exec` calls remain.
- go-git backend can perform essential operations with minimal fallback.
- Behavior is testable and consistent across backends.

## File and API References

- `workspace-manager/cmd/wsm/root.go`: root CLI wiring, Cobra commands, config initialization.
- `workspace-manager/cmd/cmds/`: command handlers mapping CLI calls to `pkg/wsm`.
- `workspace-manager/pkg/wsm/workspace.go`: workspace create/delete, worktree orchestration.
- `workspace-manager/pkg/wsm/status.go`: status aggregation and concurrency.
- `workspace-manager/pkg/wsm/git_operations.go`: workspace-wide commit and diff.
- `workspace-manager/pkg/wsm/sync_operations.go`: pull/push/rebase orchestration.
- `workspace-manager/pkg/wsm/rebase_operations.go`: CLI-only rebase flow.
- `workspace-manager/pkg/wsm/git_utils.go`: legacy branch merge/rebase checks.
- `workspace-manager/pkg/wsm/git_integration.go`: backend selection (`WSM_GIT_BACKEND`).
- `workspace-manager/pkg/wsm/gitclient/client.go`: GitClient/WorktreeManager interfaces.
- `workspace-manager/pkg/wsm/gitclient/gogit_client.go`: go-git backend with ErrNotImplemented gaps.
- `workspace-manager/pkg/wsm/gitclient/cli_client.go`: CLI backend implementation.
- `workspace-manager/README.md`: high-level product overview and usage (note build path drift).
- `workspace-manager/IMPLEMENTATION.md`: existing architecture guide (may predate backend refactor).
