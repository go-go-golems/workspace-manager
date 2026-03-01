---
Title: Investigation Diary
Ticket: WSM-MO-002-REF-GAP1-REMOTE-BRANCH
Status: active
Topics:
    - refactor
    - architecture
    - workspace-manager
    - git
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/wsm/gitclient/cli_client.go
      Note: Investigated branch output normalization
    - Path: pkg/wsm/gitclient/gogit_client.go
      Note: Investigated local-only branch iteration
    - Path: pkg/wsm/workspace.go
      Note: Chronological evidence from worktree call chains
    - Path: ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.log
      Note: Diary experiment output
    - Path: ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.sh
      Note: Diary experiment automation
ExternalSources: []
Summary: Chronological investigation log for remote branch detection false negatives.
LastUpdated: 2026-02-28T14:26:00-05:00
WhatFor: Execution diary for future maintainers and intern handoff.
WhenToUse: Use when retracing evidence collection and reproduction steps for gap 1.
---


# Investigation Diary

## Goal

Produce a reproducible, evidence-backed bug ticket for refactor gap 1 (`CheckRemoteBranchExists` false negatives), including intern-ready root-cause explanation and fix design.

## Context

This ticket was created from the gap list in:

- `ttmp/2026/01/09/WSM-MO-001-ANALYZE-REFACTOR--analyze-workspace-manager-refactor/analysis/01-refactor-status-and-target-architecture.md`

Gap 1 statement there: remote branch detection inconsistency.

## Chronological Log

## Step 1: Locate Exact Gap Definition and Existing Findings

Command:

```bash
rg -n "gap 1|refactor gap|CheckRemoteBranchExists|HybridClient" \
  workspace-manager/ttmp/2026/01/09/WSM-MO-001-ANALYZE-REFACTOR--analyze-workspace-manager-refactor/analysis -S
```

Finding:

1. Gap 1 explicitly references remote branch detection inconsistency.
2. Existing docs already flag this as correctness risk.

## Step 2: Inspect WorkspaceManager Branch-Resolution Call Paths

Commands:

```bash
nl -ba workspace-manager/pkg/wsm/workspace.go | sed -n '240,360p'
nl -ba workspace-manager/pkg/wsm/workspace.go | sed -n '1020,1125p'
```

Findings:

1. `createWorktree` and `CreateWorktreeForAdd` both call `CheckRemoteBranchExists`.
2. Both paths ignore errors from branch queries (`branches, _ := ...`, `remoteBranchExists, _ := ...`).
3. Both paths gate `opts.RemoteBranch = "origin/" + branch` on this check.

Interpretation:

- False negatives force branch creation from local/default base instead of known remote branch.

## Step 3: Inspect Backend Semantics for `ListBranches`

Commands:

```bash
nl -ba workspace-manager/pkg/wsm/gitclient/cli_client.go | sed -n '50,70p'
nl -ba workspace-manager/pkg/wsm/gitclient/gogit_client.go | sed -n '49,60p'
```

Findings:

1. CLI strips `remotes/origin/` prefix before returning branch names.
2. go-git iterates local branches only.

Interpretation:

- `CheckRemoteBranchExists` assumption (`origin/` prefix in returned list) cannot hold reliably.

## Step 4: Build Reproduction Experiment Script

Created file:

- `scripts/repro_remote_branch_false_negative.sh`

Experiment design:

1. Create bare origin.
2. Push `main` and `feature/remote-only` branches.
3. Clone a fresh client without local feature branch.
4. Confirm remote branch exists using `git ls-remote --heads`.
5. Call actual `WorkspaceManager.CheckRemoteBranchExists` via tiny Go helper for `cli`, `gogit`, and `hybrid` modes.

## Step 5: Execute Experiment and Capture Output

Command:

```bash
cd workspace-manager
./ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.sh
```

Output log:

- `scripts/repro_remote_branch_false_negative.log`

Observed result summary:

1. Ground truth: remote branch exists.
2. Method result: `exists=false` for `cli`, `gogit`, `hybrid`.

Conclusion:

- Bug reproduced directly against current production method.

## Step 6: Fix Strategy Selection

Decision:

1. Do not patch parsing in caller.
2. Add dedicated `RemoteBranchExists` API to `GitClient`.
3. Migrate caller logic and remove ignored errors in branch-resolution path.
4. De-duplicate duplicate branch-resolution logic into shared helper.

Reasoning:

- The bug is contract-level, not merely formatting-level.

## Quick Reference

### Reproduction Commands

```bash
cd /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager
./ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.sh
sed -n '1,220p' ./ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.log
```

### Key Evidence Files

1. `pkg/wsm/workspace.go`
2. `pkg/wsm/gitclient/cli_client.go`
3. `pkg/wsm/gitclient/gogit_client.go`
4. `scripts/repro_remote_branch_false_negative.log`

## Usage Examples

### Example: Intern Baseline Re-run

```bash
cd /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager
bash ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.sh
```

Expected pre-fix line pattern:

```text
backend=cli exists=false err=<nil>
backend=gogit exists=false err=<nil>
backend=hybrid exists=false err=<nil>
```

Expected post-fix pattern (minimum):

```text
backend=cli exists=true err=<nil>
backend=hybrid exists=true err=<nil>
```

## Related

1. Design doc: `design-doc/01-bug-report-and-fix-plan-remote-branch-detection.md`
2. Parent audit ticket doc: `ttmp/2026/01/09/WSM-MO-001-ANALYZE-REFACTOR--analyze-workspace-manager-refactor/analysis/02-osha-style-code-and-architecture-review.md`

## Implementation Diary (2026-02-28, Phase 2)

### Step A: Implement explicit branch-existence APIs

Commands run:

```bash
# edited files
pkg/wsm/gitclient/client.go
pkg/wsm/gitclient/cli_client.go
pkg/wsm/gitclient/gogit_client.go
pkg/wsm/gitclient/hybrid_client.go
pkg/wsm/workspace.go
```

Changes made:

1. Added `RemoteBranchExists(...)` to `GitClient`.
2. Added backend implementations for CLI and go-git.
3. Added `HybridClient` fallback implementation for this method.
4. Reworked `WorkspaceManager.CheckRemoteBranchExists` to call explicit API.
5. Removed ignored errors in branch-resolution paths.
6. Added shared `resolveBranchState` helper and reused it in both:
   - `createWorktree`
   - `CreateWorktreeForAdd`

### Step B: Add tests and iterate on semantics bug found during testing

Commands run:

```bash
go test ./pkg/wsm/gitclient -run 'Hybrid|RemoteBranch' -v
go test ./pkg/wsm -run 'CheckRemoteBranchExists|ResolveBranchState' -v
```

Observation:

- Initial `resolveBranchState` test failed because local existence was derived from `ListBranches`, and CLI `ListBranches` includes normalized remote names.

Fix applied:

1. Added `LocalBranchExists(...)` to `GitClient` and all backends.
2. Switched `resolveBranchState` + `CheckBranchExists` to explicit local branch API.
3. Re-ran tests; all targeted tests passed.

### Step C: Reproduction script re-run (post-fix)

Command:

```bash
./ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.sh
```

Result (from log):

```text
backend=cli exists=true err=<nil>
backend=gogit exists=true err=<nil>
backend=hybrid exists=true err=<nil>
```

Interpretation:

- Gap 1 false-negative behavior is no longer reproduced.

### Step D: Broader validation

Command:

```bash
go test ./...
```

Result:

- `pkg/wsm` and `pkg/wsm/gitclient` passed.
- Integration scenarios still fail with existing stale-binary / sandbox-environment leakage behavior (`.out/wsm`, repo discovery contamination), previously known and outside this ticket's direct scope.

### Step E: Ticket closure and documentation hygiene

Commands:

```bash
docmgr doctor --ticket WSM-MO-002-REF-GAP1-REMOTE-BRANCH --stale-after 30
docmgr ticket close --ticket WSM-MO-002-REF-GAP1-REMOTE-BRANCH --changelog-entry "All gap 1 implementation tasks completed; explicit local/remote branch abstraction shipped with regression coverage."
```

Result:

- Doctor passed.
- Ticket status moved `active -> complete`.
- Tasks are fully closed.
