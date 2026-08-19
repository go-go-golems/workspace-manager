---
Title: Forked Workspace Rebase/Merge Status Bug — Analysis and Implementation Guide
Ticket: WSM-MO-013-FORK-REBASE-STATUS
Status: active
Topics:
    - workspace-manager
    - git
    - rebase
    - fork
    - bugfix
    - status
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql/.wsm/wsm.json
      Note: failing workspace manifest (baseBranch task/deploy-dev-indexer, local-only)
    - Path: repo://pkg/wsm/branch/resolver.go
      Note: resolveFromState policy mirrored by proposed ResolveBaseRef
    - Path: repo://pkg/wsm/git_utils.go
      Note: Buggy CheckBranchMerged/CheckBranchNeedsRebase (hardcoded origin/base, swallow vs return)
    - Path: repo://pkg/wsm/gitclient/cli_client.go
      Note: for-each-ref existence primitives to reuse
    - Path: repo://pkg/wsm/status.go
      Note: getRepositoryStatusWithClient call site with err==nil guards
    - Path: repo://pkg/wsm/workflows/fork_workflow.go
      Note: Plan detects base from current branch -> local-only base on fork
ExternalSources: []
Summary: Intern-oriented analysis, design, and implementation guide for the forked-workspace status bug where the merge/rebase checks assume the configured base branch exists as a remote-tracking ref (origin/<base>) and silently misreport status when it does not.
LastUpdated: 2026-08-19T10:30:00-04:00
WhatFor: Onboarding an unfamiliar engineer to the WSM status pipeline, the branch-resolution subsystem, and the specific bug where forked workspaces misreport IsMerged/NeedsRebase.
WhenToUse: Read this before touching pkg/wsm/git_utils.go, pkg/wsm/status.go, or pkg/wsm/branch/.
---







# Forked Workspace Rebase/Merge Status Bug — Analysis and Implementation Guide

> Audience: a new engineer (intern-level) joining the `workspace-manager` (WSM)
> project. This document assumes you can read Go and use `git` on the command
> line, but assumes **no** prior knowledge of this codebase. It explains every
> layer you need to understand, then walks through the bug, the fix, and how to
> validate it.

## 1. Executive Summary

`wsm status` walks every repository in a workspace and reports two derived
booleans per repo: **IsMerged** ("is my branch already merged into the base?")
and **NeedsRebase** ("is my branch behind the base and needs rebasing?").

Today, both checks are computed against a single hardcoded Git reference:
`origin/<baseBranch>` — the *remote-tracking* ref on the default remote
`origin`. The code assumes that reference always exists.

That assumption breaks for **forked workspaces**. When a workspace is forked
from a base branch that only exists **locally** (a task branch that was never
pushed to `origin`), there is no `origin/<baseBranch>` to compare against. In
that situation:

- The **merge** check silently fails and reports `IsMerged = false` even though
  it actually could not determine the answer. This is a *false negative*.
- The **rebase** check fails with `exit status 128` and the error is swallowed
  by the caller, leaving `NeedsRebase` at its default `false`. This is a *false
  negative / unknown*.

The observable symptom is exactly the debug log the user reported:

```
DBG Branch merge check result ... merged=false ...
DBG Failed to check for commits ahead on configured remote base error="exit status 128"
```

The root cause is in **`pkg/wsm/git_utils.go`**: `CheckBranchMerged` and
`CheckBranchNeedsRebase` build the ref `origin/<base>` unconditionally and never
verify it exists, and they do not fall back to the local base branch when the
remote-tracking ref is missing. The codebase already has a clean abstraction for
exactly this situation — the `branch` package's `BranchResolutionPlan` — but
these two functions bypass it.

This document explains the whole status pipeline, proves the bug from
repository evidence, proposes a fix that reuses the existing branch-resolution
abstraction (preferring remote-tracking ref, falling back to local base,
marking "unknown" otherwise), and gives a phased implementation plan with a test
strategy that reproduces the forked-workspace scenario.

## 2. Problem Statement and Scope

### 2.1 The user-reported symptom

Running `wsm status` in the forked workspace
`/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql` produced, for each
sub-repo (`geppetto`, `sessionstream`, `flowkit`, `rag-ttc`), the same pattern:

```
DBG Checking if branch is merged to configured remote base
    base_branch=task/deploy-dev-indexer branch=task/ragkit-coinvault-mysql
    path=.../geppetto upstream=origin/task/deploy-dev-indexer
DBG Branch merge check result ... merged=false
DBG Checking if branch needs rebase on configured remote base ... upstream=origin/task/deploy-dev-indexer
DBG Failed to check for commits ahead on configured remote base error="exit status 128"
```

Two things are wrong and worth separating:

1. `merged=false` is reported as a confident result even though the underlying
   git command failed.
2. The rebase check *returns an error* (`exit status 128`) instead of a result.

### 2.2 In scope

- Why the remote-tracking ref `origin/<base>` is missing for forked workspaces.
- Why the merge and rebase checks behave differently (one swallows, one
  returns) and why both are wrong.
- The fix in `pkg/wsm/git_utils.go` and the call site in `pkg/wsm/status.go`.
- A regression test that builds a forked-workspace-shaped fixture (base branch
  local-only, never pushed).

### 2.3 Out of scope

- The *worktree branch reuse* and *rebase-in-progress detection* bugs already
  fixed in ticket **WSM-MO-012-WORKTREE-REBASE-BUGFIX** (different root causes:
  `-b` on existing branches and `.git` indirection in worktrees).
- Changing the on-disk workspace format (`wsm.json`).
- Changing the public CLI flags of `wsm status`.

## 3. Current-State Architecture (with evidence)

This section is the "how does this system even work" orientation. Read it once
end to end before looking at the bug.

### 3.1 The four-layer model

WSM is a deliberately layered Go application. From the architecture doc
(`pkg/docs/04-architecture-overview.md`):

```
┌──────────────────────────────────────────────────┐
│  cmd/wsm/cmds/          CLI adapters             │
│    registry/   workspace/   git/   js/           │
├──────────────────────────────────────────────────┤
│  pkg/wsm/workflows/     Orchestration layer      │
│    create, status, commit, rebase, fork, ...     │
├──────────────────────────────────────────────────┤
│  pkg/wsm/               Core domain services     │
│    workspace, discovery, git_operations, branch/ │
├──────────────────────────────────────────────────┤
│  pkg/wsm/gitclient/     Git abstraction          │
│    cli backend (system git)                      │
└──────────────────────────────────────────────────┘
```

Dependencies flow **downward only**. A workflow may call a core domain service
and the git client; the git client never calls a workflow. This matters for the
fix: we want to keep the resolution logic in the core layer (`pkg/wsm/branch`),
not push git heuristics into the workflow layer.

### 3.2 The status call chain

When you run `wsm status`, control flows like this:

```
cmd/wsm/cmds/workspace/  ──►  workflows.StatusWorkflow.GetStatus
                                   │
                                   ▼
                          wsm.StatusChecker.GetWorkspaceStatusWithOptions
                                   │  (per repo, concurrent via errgroup)
                                   ▼
                          getRepositoryStatusWithClient
                                   │
                 ┌─────────────────┼──────────────────┐
                 ▼                 ▼                  ▼
          gc.Status         gc.AheadBehind     CheckBranchMerged /
          (porcelain)      (HEAD...@{up})     CheckBranchNeedsRebase
                                                       │
                                                       ▼
                                            branchsvc.ResolveBaseBranch +
                                            branchsvc.RemoteTrackingRef
                                            (pkg/wsm/branch)
                                                       │
                                                       ▼
                                               exec "git ..."  ← runs system git
```

Key files (follow these in order to read the code):

1. `pkg/wsm/workflows/status_workflow.go` — `StatusWorkflow.GetStatus`:
   resolves the workspace name (from flag or CWD), loads the workspace, calls
   the checker.
2. `pkg/wsm/status.go` — `StatusChecker.GetWorkspaceStatusWithOptions`:
   iterates repositories, optionally concurrently with an `errgroup` + weighted
   semaphore, and assembles a `WorkspaceStatus`.
3. `pkg/wsm/status.go` — `getRepositoryStatusWithClient`: the per-repo
   computation. This is where `IsMerged` and `NeedsRebase` get set.
4. `pkg/wsm/git_utils.go` — `CheckBranchMerged` / `CheckBranchNeedsRebase`:
   the functions under the microscope.
5. `pkg/wsm/branch/` — the branch-resolution subsystem (types, resolver,
   service). The *correct* place for "which ref should I compare against"
   logic.

### 3.3 What a workspace *is*

A workspace is a directory containing git worktrees for several repos, plus a
small JSON manifest. From `pkg/wsm/types.go`:

```go
// pkg/wsm/types.go:25-33
type Workspace struct {
    Name         string       `json:"name"`
    Path         string       `json:"path"`
    Repositories []Repository `json:"repositories"`
    Branch       string       `json:"branch"`      // the workspace's own branch
    BaseBranch   string       `json:"base_branch"` // what it was forked from
    Created      time.Time    `json:"created"`
    GoWorkspace  bool         `json:"go_workspace"`
    AgentMD      string       `json:"agent_md"`
}
```

The manifest for the failing workspace lives at
`/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql/.wsm/wsm.json`. The
two fields that matter for this bug are:

- `"branch": "task/ragkit-coinvault-mysql"` — the branch every repo is checked
  out on (the fork's own branch).
- `"baseBranch": "task/deploy-dev-indexer"` — the branch the fork was created
  from. **This is the value passed as `baseBranch` into the status checks.**

### 3.4 The per-repo status record

```go
// pkg/wsm/types.go:46-58
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
    IsMerged       bool       `json:"is_merged"`    // ← silently wrong on fork
    NeedsRebase    bool       `json:"needs_rebase"` // ← silently unknown on fork
}
```

Note the comments: both fields claim to compare against "the default remote main
branch". That comment itself encodes the buggy assumption — the comparison ref
is not necessarily a *remote* ref at all.

### 3.5 How the two checks are invoked

In `getRepositoryStatusWithClient` (`pkg/wsm/status.go:119-160`):

```go
// pkg/wsm/status.go:151-157
// Preserve legacy semantics used by status table columns.
if isMerged, err := CheckBranchMerged(ctx, repoPath, baseBranch); err == nil {
    status.IsMerged = isMerged
}
if needsRebase, err := CheckBranchNeedsRebase(ctx, repoPath, baseBranch); err == nil {
    status.NeedsRebase = needsRebase
}
```

Both calls use the **`err == nil` guard**, which means: *if the check returns
an error, silently keep the zero value*. This is the second half of the bug —
the caller is designed to tolerate errors by doing nothing, which converts an
"unknown" into a confident-looking `false`.

### 3.6 The branch-resolution subsystem (the abstraction we should reuse)

This is the most important section for the fix. WSM already has a typed
subsystem for deciding *which* git ref to operate on, in `pkg/wsm/branch/`.

**Types** (`pkg/wsm/branch/types.go`):

```go
// pkg/wsm/branch/types.go:35-49
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
    ResolutionStrategyUseLocal        // refs/heads/<branch> exists → use it
    ResolutionStrategyTrackRemote     // refs/remotes/<remote>/<branch> exists
    ResolutionStrategyCreateFromBase  // neither exists; create from base
    ResolutionStrategyCreateFromHead  // neither exists; create from HEAD
)
```

**Ref builders** (`pkg/wsm/branch/types.go`):

```go
// RemoteTrackingRef builds "<remote>/<branch>" using typed domain values.
func RemoteTrackingRef(remote RemoteName, branch BranchName) string { ... }

// ResolveBaseBranch returns the configured base branch with a safe default.
// Priority: explicit arg → WSM_BASE_BRANCH env → "main".
func ResolveBaseBranch(explicit string) BranchName { ... }
```

**The resolution policy** (`pkg/wsm/branch/resolver.go`):

```go
// resolveFromState picks a strategy from observed ref existence.
func resolveFromState(req, defaultRemote, localExists, remoteTrackingExists) {
    switch {
    case localExists:
        return ResolutionStrategyUseLocal     // use the local branch ref
    case remoteTrackingExists:
        return ResolutionStrategyTrackRemote   // use origin/<branch>
    case req.BaseBranch != "":
        return ResolutionStrategyCreateFromBase
    default:
        return ResolutionStrategyCreateFromHead
    }
}
```

**The service** (`pkg/wsm/branch/service_impl.go`) wires this to the git
client and exposes `Resolve(ctx, repoPath, req) (*BranchResolutionPlan, error)`,
which checks `LocalBranchExists` and `RemoteTrackingBranchExists` and returns a
plan describing exactly which ref to use.

The crucial point: **the status checks in `git_utils.go` do not use any of
this.** They call `ResolveBaseBranch` and `RemoteTrackingRef` directly and then
hard-execute git against the resulting `origin/<base>` string. The resolution
subsystem exists precisely to avoid that, but the two legacy status functions
predate it (or were never migrated). The fix is to route them through the same
resolution path.

### 3.7 The git client primitives we need

The CLI git backend (`pkg/wsm/gitclient/cli_client.go`) already implements the
two existence checks the fix needs:

```go
// pkg/wsm/gitclient/client.go (interface)
LocalBranchExists(ctx, repo, branch) (bool, error)
//   → for-each-ref refs/heads/<branch>

RemoteTrackingBranchExists(ctx, repo, remote, branch) (bool, error)
//   → for-each-ref refs/remotes/<remote>/<branch>
```

And `AheadBehind(ctx, repo, upstream)` which, given an explicit `upstream`,
runs `git rev-list --left-right --count HEAD...<upstream>` and returns
`(ahead, behind, err)`. This is the *non-buggy* version of what
`CheckBranchNeedsRebase` hand-rolls.

## 4. Gap Analysis (evidence-backed)

### 4.1 Gap 1 — the remote-tracking ref does not exist in the fork

Reproduced on the actual failing repo
(`/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql/geppetto`):

```
$ git remote -v
origin  git@github.com:go-go-golems/geppetto.git
wesen   git@github.com:wesen/geppetto
$ git rev-list --count HEAD..origin/task/deploy-dev-indexer
fatal: ambiguous argument 'HEAD..origin/task/deploy-dev-indexer': unknown revision
$ echo $?   → 128
$ git merge-base --is-ancestor HEAD origin/task/deploy-dev-indexer
fatal: Not a valid object name origin/task/deploy-dev-indexer
$ echo $?   → 128
```

And the decisive ref-existence checks:

```
$ git for-each-ref refs/remotes/origin/task/deploy-dev-indexer
   (empty — the remote-tracking ref DOES NOT exist)

$ git for-each-ref refs/heads/task/deploy-dev-indexer
   335a807a... commit refs/heads/task/deploy-dev-indexer
   (the LOCAL base branch DOES exist)
```

So the configured base branch `task/deploy-dev-indexer` is a **local-only task
branch** — it was never pushed to `origin`. There is simply nothing at
`origin/task/deploy-dev-indexer` to compare against.

### 4.2 Gap 2 — the merge check silently returns a false `false`

In `CheckBranchMerged` (`pkg/wsm/git_utils.go:40-52`):

```go
cmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", "HEAD", remoteBase)
cmd.Dir = path
err := cmd.Run()
merged := err == nil        // ← exit 128 → err != nil → merged = false
...
return merged, nil           // ← error DISCARDED; returns (false, nil)
```

Because `merge-base --is-ancestor` exits `0` only when HEAD is an ancestor of
the base (i.e. "merged"), *any* failure — including "the ref doesn't exist" —
is interpreted as "not merged". The function returns `(false, nil)`, so the
caller's `err == nil` guard passes and `IsMerged` is set to `false`.

This is a logic error: *failure to check* must not be reported as *checked and
negative*.

### 4.3 Gap 3 — the rebase check returns an error that the caller swallows

In `CheckBranchNeedsRebase` (`pkg/wsm/git_utils.go:83-92`):

```go
cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", "HEAD.."+remoteBase)
cmd.Dir = path
output, err := cmd.Output()
if err != nil {
    log.Debug().Err(err)...Msg("Failed to check for commits ahead on configured remote base")
    return false, err          // ← returns the 128 error
}
```

Unlike the merge check, this one *does* return the error. But the caller
(`pkg/wsm/status.go:155`) only assigns `NeedsRebase` when `err == nil`, so on
error `NeedsRebase` stays at its zero value `false`. The user sees "no rebase
needed" when the truth is "I couldn't tell".

### 4.4 Why forked workspaces hit this and normal ones don't

A normal workspace is created from a base branch that *is* a remote-tracking
ref (e.g. `origin/main`). After `git fetch`, `origin/main` exists locally as a
remote-tracking ref, so both checks work.

A **forked** workspace is created by `ForkWorkflow` (`pkg/wsm/workflows/fork_workflow.go`).
Its `Plan` method detects the base branch by reading the *current branch* of
the source workspace's repositories (`fork_workflow.go:71-88`):

```go
status, err := fw.checker.GetWorkspaceStatus(ctx, sourceWorkspace)
...
baseBranch := status.Repositories[0].CurrentBranch
```

So the fork's `baseBranch` is whatever branch the source was on — frequently a
*local task branch* that was never pushed. The fork then creates worktrees on a
new branch from that base. The new workspace's repos are on the new branch, but
the base they were cut from has no `origin/<base>` ref. Result: the very status
check the fork workflow itself used to *detect* the base branch will, once the
workspace is forked, start failing for the fork's own status — because the
comparison ref no longer exists remotely.

## 5. Proposed Architecture and APIs

### 5.1 Goal

Make `IsMerged` and `NeedsRebase` **correct** and **honest**:

- Correct: compare against a ref that actually exists (prefer
  `origin/<base>`, fall back to the local `<base>`).
- Honest: when no usable ref exists, say "unknown" instead of a confident
  `false`.

### 5.2 Introduce an explicit "unknown" state

Today `IsMerged`/`NeedsRebase` are bare `bool`, where `false` is overloaded to
mean both "checked: no" and "couldn't check". We disambiguate with a small
typed result. Keep the JSON shape backward compatible by leaving `false` as the
default observable value.

```go
// pkg/wsm/types.go (add near RepositoryStatus)

// MergeRebaseStatus is a tri-state: Unknown means the comparison ref
// could not be resolved (e.g. forked workspace with local-only base).
type MergeRebaseStatus int

const (
    // MergeStatusUnknown keeps JSON "is_merged"/"needs_rebase" at false
    // for backward compatibility, but records that no real check ran.
    MergeRebaseUnknown MergeRebaseStatus = iota
    MergeRebaseYes
    MergeRebaseNo
)
```

> **Compatibility note:** Do not change the existing `IsMerged bool` /
> `NeedsRebase bool` JSON fields or the status-table columns in this ticket.
> Add the tri-state as internal plumbing and a new optional debug field
> (e.g. `base_ref_resolved string` / `base_ref_status string`) so the table can
> one day show "unknown". The minimum viable fix keeps the existing fields but
> makes `false` mean "checked: no" rather than "couldn't check", and surfaces
> "unknown" in a new column/field. See Decision D2.

### 5.3 A reusable ref resolver for status checks

Add a small helper in `pkg/wsm/branch` (or `pkg/wsm`) that, given a repo path
and a configured base branch, returns the concrete ref to compare against —
reusing the existing existence primitives.

```go
// ResolveBaseRef picks the concrete git ref to compare HEAD against for
// merge/rebase status. Preference order:
//   1. refs/remotes/<remote>/<base>  (remote-tracking)  → "origin/<base>"
//   2. refs/heads/<base>             (local branch)     → "<base>"
//   3. none
func ResolveBaseRef(ctx context.Context, gc gitclient.GitClient,
    repoPath string, base branch.BranchName, remote branch.RemoteName,
) (ref string, found bool, err error)
```

Pseudocode:

```text
func ResolveBaseRef(gc, repoPath, base, remote):
    h = gc.Open(repoPath)
    if remote == "": remote = "origin"
    # 1) prefer remote-tracking ref
    if gc.RemoteTrackingBranchExists(h, remote, base):
        return remote + "/" + base, true, nil
    # 2) fall back to local base branch
    if gc.LocalBranchExists(h, base):
        return base, true, nil
    # 3) nothing to compare against
    return "", false, nil
```

### 5.4 Rewrite the two checks to use it

`CheckBranchMerged` and `CheckBranchNeedsRebase` become thin wrappers around
`ResolveBaseRef` plus the existing git plumbing, with **no silent swallowing**:

```go
// CheckBranchMerged returns (merged, resolved bool, err).
// resolved=false means no usable base ref was found (forked workspace,
// local-only base, etc.) — caller must treat as "unknown", NOT as "no".
func CheckBranchMerged(ctx, repoPath, baseBranch) (merged, resolved bool, err error)
func CheckBranchNeedsRebase(ctx, repoPath, baseBranch) (needsRebase, resolved bool, err error)
```

### 5.5 Update the call site to honor "unknown"

In `getRepositoryStatusWithClient`, distinguish "unknown" from "no":

```go
// pkg/wsm/status.go (proposed)
if merged, resolved, err := CheckBranchMerged(ctx, repoPath, baseBranch); err == nil {
    if !resolved {
        status.IsMerged = false            // backward-compat default
        status.BaseRefStatus = "unknown"   // NEW field
    } else {
        status.IsMerged = merged
        status.BaseRefStatus = "resolved"
    }
}
// same shape for NeedsRebase
```

The status table gains a column showing the resolved base ref (or "unknown"),
so a human can immediately see *why* IsMerged/NeedsRebase are what they are.

## 6. Decision Records

### Decision: D1 — Compare against the resolved base ref, not always `origin/<base>`

- **Context:** The base branch of a forked workspace is frequently a local-only
  task branch. Hardcoding `origin/<base>` makes status fail for exactly the
  workspaces WSM exists to support (forks of in-flight work).
- **Options considered:**
  1. Always `origin/<base>` (status quo) — fails for forks.
  2. Always local `<base>` — wrong for normal workspaces where the
    remote-tracking ref is more up to date than the stale local branch.
  3. Prefer `origin/<base>`, fall back to local `<base>` — correct for both.
- **Decision:** Option 3. Reuse `branch.RemoteTrackingBranchExists` +
  `LocalBranchExists` to pick the ref.
- **Rationale:** Matches the existing `BranchResolutionPlan` philosophy
  (`resolver.go`: localExists → use-local; else remoteTrackingExists → track).
  For *status* we invert the preference (remote first, because it reflects the
  shared truth), but reuse the same existence primitives.
- **Consequences:** Status now works for forks. Slightly more git calls
  (two `for-each-ref` per repo, cheap). Must document the preference order.
- **Status:** proposed

### Decision: D2 — "Unknown" is a first-class status, not a silent `false`

- **Context:** The merge check today reports `false` on failure; the rebase
  check today returns an error the caller swallows into `false`. Both produce a
  confident-looking "no" that is actually "I don't know".
- **Options considered:**
  1. Keep bare `bool`, fix the git commands, accept that "no ref" still means
    `false`.
  2. Add a tri-state internally + a new optional field/column for "unknown",
    keep existing `bool` JSON for compatibility.
  3. Change `IsMerged`/`NeedsRebase` to `*bool` (nullable) in JSON — breaks
    consumers.
- **Decision:** Option 2. Internal `MergeRebaseStatus` + new optional
  `base_ref_status`/`base_ref` fields. Existing `is_merged`/`needs_rebase`
  remain `bool` and default `false`.
- **Rationale:** Honest without breaking the JSON contract or the status table
  columns that already exist. The new column is additive.
- **Consequences:** Must thread `resolved bool` out of the check functions and
  into the caller. Adds one column to the human status table (gated, can be
  hidden).
- **Status:** proposed

### Decision: D3 — Fix lives in `pkg/wsm/git_utils.go` + `pkg/wsm/branch`, not the workflow layer

- **Context:** The bug could be "fixed" in the workflow by catching the error.
  But that hides the real problem (wrong ref) and duplicates resolution logic.
- **Options considered:**
  1. Workflow-layer try/catch around the existing calls.
  2. New `ResolveBaseRef` in `pkg/wsm/branch`, used by the two check functions.
- **Decision:** Option 2. Keep resolution in the core/branch layer per the
  four-layer model; workflows stay thin.
- **Rationale:** The `branch` package already owns "which ref" policy. Adding a
  status-oriented resolver there is consistent and testable in isolation.
- **Consequences:** `git_utils.go` gains a dependency on `gitclient` (it
  currently shells out directly). Acceptable — `status.go` already holds a
  `gitclient.GitClient` and can pass it down.
- **Status:** proposed

## 7. Pseudocode and Key Flows

### 7.1 Current (buggy) flow

```text
getRepositoryStatusWithClient(repo, repoPath, baseBranch):
    handle = gc.Open(repoPath)
    st = gc.Status(handle)
    ...
    if CheckBranchMerged(repoPath, baseBranch) returns (merged, nil):
        status.IsMerged = merged          # ← bug: (false, nil) on missing ref

CheckBranchMerged(repoPath, baseBranch):
    base = ResolveBaseBranch(baseBranch)              # "task/deploy-dev-indexer"
    remoteBase = RemoteTrackingRef("origin", base)    # "origin/task/deploy-dev-indexer"
    err = git merge-base --is-ancestor HEAD remoteBase
    return (err == nil, nil)                          # ← exit 128 ⇒ (false, nil)

CheckBranchNeedsRebase(repoPath, baseBranch):
    remoteBase = "origin/task/deploy-dev-indexer"
    out, err = git rev-list --count HEAD..remoteBase
    if err != nil: return (false, err)                # ← exit 128 ⇒ error swallowed upstream
    return (out != "0", nil)
```

### 7.2 Proposed flow

```text
getRepositoryStatusWithClient(repo, repoPath, baseBranch, gc):
    handle = gc.Open(repoPath)
    st = gc.Status(handle)
    ...
    merged, resolved, err = CheckBranchMerged(ctx, gc, repoPath, baseBranch)
    if err == nil:
        status.IsMerged = merged
        status.BaseRefResolved = resolved
        status.BaseRef = lastResolvedRef          # for the table column
    needsRebase, resolved, err = CheckBranchNeedsRebase(ctx, gc, repoPath, baseBranch)
    if err == nil:
        status.NeedsRebase = needsRebase
        status.BaseRefResolved = status.BaseRefResolved && resolved

CheckBranchMerged(ctx, gc, repoPath, baseBranch):
    base = ResolveBaseBranch(baseBranch)
    ref, ok, err = ResolveBaseRef(ctx, gc, repoPath, base, "origin")
    if err != nil: return (false, false, err)
    if !ok:    return (false, false, nil)          # UNKNOWN — caller must see resolved=false
    err = git merge-base --is-ancestor HEAD ref
    return (err == nil, true, nil)                # resolved=true now

CheckBranchNeedsRebase(ctx, gc, repoPath, baseBranch):
    base = ResolveBaseBranch(baseBranch)
    ref, ok, err = ResolveBaseRef(ctx, gc, repoPath, base, "origin")
    if err != nil: return (false, false, err)
    if !ok:    return (false, false, nil)          # UNKNOWN
    out, err = git rev-list --count HEAD..ref
    if err != nil: return (false, false, err)
    return (strings.TrimSpace(out) != "0", true, nil)
```

### 7.3 Resolution preference diagram

```text
         configured base branch (e.g. task/deploy-dev-indexer)
                              │
                              ▼
                ┌──────────────────────────────┐
                │  refs/remotes/origin/<base>?  │── yes ──► use "origin/<base>"
                └──────────────────────────────┘
                              │ no
                              ▼
                ┌──────────────────────────────┐
                │  refs/heads/<base>?           │── yes ──► use "<base>"
                └──────────────────────────────┘
                              │ no
                              ▼
                    resolved = false  (UNKNOWN)
```

## 8. Implementation Phases

Each phase is independently testable and committable.

### Phase 1 — Add `ResolveBaseRef` and unit-test it

- File: `pkg/wsm/branch/` (new `status_resolve.go`) or `pkg/wsm/git_utils.go`.
- Implement `ResolveBaseRef(ctx, gc, repoPath, base, remote) (ref, found, err)`
  using `gc.RemoteTrackingBranchExists` then `gc.LocalBranchExists`.
- Unit test with a fixture that has a local-only base branch (no
  remote-tracking ref) and asserts `found=true, ref=<base>`; plus a fixture
  with the remote-tracking ref and asserts `ref=origin/<base>`; plus a fixture
  with neither and asserts `found=false`.

### Phase 2 — Rewrite the two checks to be honest

- File: `pkg/wsm/git_utils.go`.
- Change signatures to return `(value, resolved bool, err error)`. Accept a
  `gitclient.GitClient` (or keep shelling out via `exec` but use `ResolveBaseRef`
  for the ref). Prefer taking `gc` so existence checks use the same backend.
- Replace the hardcoded `RemoteTrackingRef(...)` with `ResolveBaseRef(...)`.
- On `!resolved`, return `(false, false, nil)` (unknown) — **do not swallow**.
- Remove the silent `merged := err == nil` conflation: a git *command* error
  is now returned as `err`, distinct from "ref missing" (resolved=false).

### Phase 3 — Thread "resolved" into the status record

- File: `pkg/wsm/types.go` — add `BaseRefResolved bool` and `BaseRef string`
  (and/or the `MergeRebaseStatus` tri-state internally). Keep `IsMerged` /
  `NeedsRebase` as `bool` for JSON compatibility.
- File: `pkg/wsm/status.go:151-157` — update the two call sites to record
  `resolved` and, when `!resolved`, optionally set `BaseRefStatus="unknown"`.

### Phase 4 — Surface "unknown" in the status table

- File: the status table formatter (search `cmd/wsm/cmds/workspace/` for the
  status command's table columns). Add an optional column `BASE_REF` showing the
  resolved ref or `unknown`. This lets a human immediately diagnose a forked
  workspace instead of being misled by `merged=false`.

### Phase 5 — Regression test reproducing the forked-workspace scenario

- File: `pkg/wsm/..._test.go`.
- Build a fixture:
  1. A bare "origin" remote; clone; commit on `main`; push `main` only.
  2. Create a local task branch `task/deploy-dev-indexer` **without pushing**.
  3. From it, create a worktree (or just a branch) `task/fork-branch`.
- Assert `wsm status` on this shape yields `BaseRefResolved=true` with
  `BaseRef="task/deploy-dev-indexer"` (the local fallback), and that
  `IsMerged`/`NeedsRebase` reflect a real comparison (not a swallowed error).
- Add the negative case: base branch that exists **nowhere** →
  `BaseRefResolved=false`, and `IsMerged`/`NeedsRebase` are `false` with
  `BaseRefStatus="unknown"`.

### Phase 6 — Validate against the real failing workspace

- Run `wsm status` against
  `/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql`.
- Confirm the new column shows `task/deploy-dev-indexer` (local fallback) and
  that `IsMerged`/`NeedsRebase` are now real booleans, not swallowed errors.
- Confirm the debug log no longer emits `exit status 128` for these repos.

## 9. Test Strategy

- **Unit (branch resolver):** `ResolveBaseRef` with three fixtures (remote,
  local-only, neither). Covers the decision matrix.
- **Unit (checks):** `CheckBranchMerged`/`CheckBranchNeedsRebase` return
  `resolved=false` for missing ref; return real values when ref resolves; return
  `err` on genuine git failure (e.g. corrupt repo).
- **Integration (forked-workspace shape):** the Phase 5 fixture, asserting the
  end-to-end `RepositoryStatus` fields.
- **Regression against the bug:** a test named `TestStatus_ForkedWorkspace_LocalOnlyBase`
  that fails on the current code (asserts `exit status 128` is gone / `resolved`
  is true via local fallback) and passes after the fix.
- **Manual:** run `wsm status` on the real workspace and on a normal
  `origin/main`-based workspace; diff the output to ensure no regression for the
  common case.

Run the suite with:

```bash
go test ./pkg/wsm/... -count=1 -run 'Status|Branch|Rebase|Merged'
go test ./... -count=1
golangci-lint run ./pkg/wsm/...
```

## 10. Risks, Alternatives, Open Questions

### Risks

- **Behavior change for normal workspaces:** if a workspace's local `<base>`
  diverges from `origin/<base>`, preferring remote first (D1) keeps current
  behavior. If neither exists, we now report "unknown" where before we reported
  a confident `false` — this is the intended honesty fix but could surprise a
  caller parsing JSON. Mitigated by keeping `is_merged`/`needs_rebase` as `bool`
  defaulting `false`.
- **Concurrency:** `GetWorkspaceStatusWithOptions` runs repos concurrently.
  `ResolveBaseRef` only reads refs (no mutation), so it is safe under the
  existing `errgroup`.
- **Extra git calls:** two `for-each-ref` per repo on top of existing calls.
  Negligible; `for-each-ref` is local and fast.

### Alternatives considered

- **Auto-fetch the base before checking.** Could `git fetch origin <base>` to
  materialize `origin/<base>`? Rejected: the base may be local-only by design
  (a task branch never pushed), so fetching would 404 or fetch the wrong thing,
  and would mutate the user's remote-tracking refs as a side effect of a
  read-only status command.
- **Store the base ref's SHA at fork time.** Persist the base commit in
  `wsm.json` at fork, compare against that SHA. More robust to ref movement but
  a bigger change to the on-disk format; defer to a follow-up.

### Open questions

- Should the human status table show `BASE_REF`/`UNKNOWN` by default, or only
  with `--verbose`? (Lean: always show `unknown`, hide the ref unless verbose.)
- Should `wsm fork` warn at fork time when the detected base branch has no
  remote-tracking ref? (Likely yes — cheap, prevents the surprise later.)

## 11. References (key files)

| File | Why it matters |
| --- | --- |
| `pkg/wsm/git_utils.go:24-52` | `CheckBranchMerged` — swallows error, returns false |
| `pkg/wsm/git_utils.go:56-101` | `CheckBranchNeedsRebase` — returns error on missing ref |
| `pkg/wsm/status.go:119-160` | `getRepositoryStatusWithClient` — the call site with `err == nil` guards |
| `pkg/wsm/types.go:46-58` | `RepositoryStatus` — `IsMerged`/`NeedsRebase` fields and their comments |
| `pkg/wsm/types.go:25-33` | `Workspace` — `BaseBranch` field read from `wsm.json` |
| `pkg/wsm/branch/types.go` | `RemoteTrackingRef`, `ResolveBaseBranch`, resolution strategies |
| `pkg/wsm/branch/resolver.go` | `resolveFromState` — the existing resolution policy to mirror |
| `pkg/wsm/branch/service_impl.go` | `Service.Resolve` — the existing `LocalBranchExists`/`RemoteTrackingBranchExists` wiring |
| `pkg/wsm/gitclient/client.go` | `GitClient` interface: `RemoteTrackingBranchExists`, `LocalBranchExists`, `AheadBehind` |
| `pkg/wsm/gitclient/cli_client.go:107-126` | CLI impls of the existence checks (`for-each-ref`) |
| `pkg/wsm/workflows/fork_workflow.go:52-88` | `ForkWorkflow.Plan` — how `baseBranch` is detected (current branch) |
| `pkg/wsm/workflows/status_workflow.go` | `StatusWorkflow.GetStatus` — entry from the CLI |
| `pkg/docs/04-architecture-overview.md` | The four-layer model used in §3.1 |
| `/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql/.wsm/wsm.json` | The failing workspace manifest (`baseBranch: task/deploy-dev-indexer`) |

### Glossary

- **Workspace:** a directory of git worktrees for several repos + a `wsm.json`
  manifest.
- **Base branch:** the branch a workspace (or fork) was created from; stored as
  `Workspace.BaseBranch`.
- **Remote-tracking ref:** a local cache of a remote branch, e.g.
  `origin/main` (lives under `refs/remotes/origin/`).
- **Forked workspace:** a workspace created by `wsm fork` from an existing
  workspace, on a new branch cut from the source's current branch.
- **`exit status 128`:** git's generic "I could not resolve this revision" exit
  code; here it means the named ref does not exist.
