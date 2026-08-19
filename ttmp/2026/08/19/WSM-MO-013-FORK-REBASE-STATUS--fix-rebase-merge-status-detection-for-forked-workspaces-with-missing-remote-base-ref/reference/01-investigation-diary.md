---
Title: Investigation Diary
Ticket: WSM-MO-013-FORK-REBASE-STATUS
Status: active
Topics:
    - workspace-manager
    - git
    - rebase
    - fork
    - bugfix
    - status
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological investigation of the forked-workspace rebase/merge status bug, from log triage to root cause and proposed fix.
LastUpdated: 2026-08-19T10:32:00.000000000-04:00
WhatFor: "Resuming or reviewing the investigation of WSM-MO-013."
WhenToUse: "Read before continuing work on the forked-workspace status bug."
---

# Investigation Diary

## Goal

Record how the `wsm status` forked-workspace bug was triaged, root-caused, and
scoped into a design + implementation plan in ticket WSM-MO-013-FORK-REBASE-STATUS.

## Step 1: Triage the user-reported log and locate the failing code

The user ran `wsm status` in the forked workspace
`/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql` and pasted debug logs
showing, per sub-repo, `merged=false` followed by
`Failed to check for commits ahead on configured remote base error="exit status 128"`
with `upstream=origin/task/deploy-dev-indexer`.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket for handling the fact that wsm cannot apparently figure out the rebase merge status when a wsm project has been forked, for exampke in /home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql" (plus the debug log block and "when I do wsm status").

**Assistant interpretation:** Create a docmgr ticket investigating why WSM cannot determine rebase/merge status for forked workspaces; reproduce and root-cause it.

**Inferred user intent:** Stop `wsm status` from silently misreporting `IsMerged`/`NeedsRebase` for forked workspaces, and document the fix for a new engineer.

### What I did
- Searched the codebase for the log strings: `rg -n "Failed to check for commits ahead" pkg` → `pkg/wsm/git_utils.go:91`.
- Confirmed the two check functions live in `pkg/wsm/git_utils.go` (`CheckBranchMerged`, `CheckBranchNeedsRebase`).
- Located the call site: `pkg/wsm/status.go:151-157` (`getRepositoryStatusWithClient`).

### Why
The log strings are the fastest path from symptom to source. The `exit status 128`
plus `upstream=origin/<base>` immediately suggested a missing git ref.

### What worked
- Direct string search pinpointed the exact file and lines in one query.
- Reading `git_utils.go` showed both checks hardcode `origin/<base>` via
  `branchsvc.RemoteTrackingRef(branchsvc.DefaultRemoteName, base)`.

### What didn't work
- Nothing failed; the investigation was straightforward.

### What I learned
- The merge check (`git_utils.go:42-52`) discards the command error and returns
  `(false, nil)`, so a missing ref looks like "not merged".
- The rebase check (`git_utils.go:85-92`) returns the error, which the caller
  swallows via an `err == nil` guard, leaving `NeedsRebase` at default `false`.

### What was tricky to build
N/A (investigation only; no code changed).

### What warrants a second pair of eyes
- Confirm that the two checks really diverge in error handling (merge swallows,
  rebase returns) — this asymmetry is the crux of why the symptom is two-faced.

### What should be done in the future
- Unify the two checks' error contract in the fix (Step 3 design).

### Code review instructions
- Start at `pkg/wsm/git_utils.go:24` (`CheckBranchMerged`) and `:56`
  (`CheckBranchNeedsRebase`).
- Validate by grepping the log strings against the source.

### Technical details
- `exit status 128` = git "unknown revision / not a valid object name".

## Step 2: Reproduce against the real repo and confirm root cause

### What I did
- Inspected `/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql/.wsm/wsm.json`:
  `baseBranch: task/deploy-dev-indexer`, `branch: task/ragkit-coinvault-mysql`.
- Ran the exact failing commands in the worktree `.../geppetto`:
  - `git rev-list --count HEAD..origin/task/deploy-dev-indexer` → `fatal: ... unknown revision`, exit 128.
  - `git merge-base --is-ancestor HEAD origin/task/deploy-dev-indexer` → `fatal: Not a valid object name`, exit 128.
- Checked ref existence:
  - `git for-each-ref refs/remotes/origin/task/deploy-dev-indexer` → **empty** (remote-tracking ref does NOT exist).
  - `git for-each-ref refs/heads/task/deploy-dev-indexer` → **exists locally** (`335a807a...`).
- Checked remotes: `origin` = `go-go-golems/geppetto`, `wesen` = `wesen/geppetto`. The base branch was never pushed to `origin`.

### Why
Prove the assumption `origin/<base>` exists is false for this workspace, and that
the local base branch *does* exist (so a local fallback is viable).

### What worked
- The ref-existence checks are decisive and reproducible: remote-tracking ref
  absent, local branch present.

### What didn't work
- One bash command broke on the `%(refname)` format string with unquoted parens;
  re-ran without the format string.

### What I learned
- The base branch `task/deploy-dev-indexer` is a **local-only task branch**.
  Forking from a local-only base is the trigger condition.
- `ForkWorkflow.Plan` (`pkg/wsm/workflows/fork_workflow.go:71-88`) detects the
  base branch as the source workspace's *current branch*, which is frequently a
  local task branch — so forks of in-flight work inherently lack `origin/<base>`.

### What was tricky to build
N/A.

### What warrants a second pair of eyes
- Verify the fork-detects-base-from-current-branch claim holds for other forked
  workspaces, not just this one.

### What should be done in the future
- Consider a `wsm fork` warning when the detected base has no remote-tracking ref.

### Code review instructions
- Reproduce with the commands above in `.../ragkit-coinvault-mysql/geppetto`.

### Technical details
- `git for-each-ref <refspec>` prints nothing when the ref is absent; non-zero
  output means it exists.

## Step 3: Map the surrounding system and design the fix

### What I did
- Read the four-layer architecture doc (`pkg/docs/04-architecture-overview.md`).
- Read the status call chain: `workflows/status_workflow.go` → `status.go`
  (`StatusChecker`, `getRepositoryStatusWithClient`) → `git_utils.go`.
- Read the branch-resolution subsystem: `pkg/wsm/branch/{types,resolver,service,service_impl}.go`
  and the `gitclient.GitClient` interface + CLI impls of
  `RemoteTrackingBranchExists`/`LocalBranchExists`/`AheadBehind`.
- Confirmed the `branch` package already has a `BranchResolutionPlan` with
  strategies (`UseLocal`, `TrackRemote`, …) and existence-based resolution, but
  the two status checks bypass it.
- Wrote the design doc (`design-doc/01-…`) with executive summary, evidence,
  decision records, pseudocode, phased plan, test strategy, and file references.

### Why
The fix should reuse the existing resolution abstraction rather than add ad-hoc
git heuristics in the workflow layer. Documenting the whole pipeline makes the
ticket intern-friendly as requested.

### What worked
- The `branch` subsystem cleanly supports a `ResolveBaseRef` helper: prefer
  `origin/<base>` (remote-tracking), fall back to local `<base>`, else "unknown".

### What didn't work
- Nothing.

### What I learned
- The asymmetry between the two checks (swallow vs return) is what produces the
  two-faced symptom (`merged=false` confidently + `exit status 128` as an error).
- The caller's `err == nil` guard is what turns "unknown" into a confident
  `false` for `NeedsRebase`.

### What was tricky to build
- Designing the "unknown" state without breaking the existing JSON `bool`
  fields (D2): keep `is_merged`/`needs_rebase` as `bool` defaulting `false`, add
  an internal tri-state + an optional `base_ref`/`base_ref_status` column.

### What warrants a second pair of eyes
- The signature change `(value, resolved bool, err)` propagating through the
  call site; ensure concurrency in `GetWorkspaceStatusWithOptions` stays safe
  (it does — resolution is read-only).

### What should be done in the future
- Implement Phases 1–6 of the design doc; add the regression test
  `TestStatus_ForkedWorkspace_LocalOnlyBase`.

### Code review instructions
- Read `design-doc/01-forked-workspace-rebase-merge-status-bug-analysis-and-implementation-guide.md` §5–§8.
- Validate proposed `ResolveBaseRef` against `pkg/wsm/branch/resolver.go`.

### Technical details
- Reuse `gitclient.RemoteTrackingBranchExists` (`cli_client.go:107`) and
  `LocalBranchExists` (`cli_client.go:53`), both `for-each-ref` based.
