---
Title: Investigation Diary
Ticket: WSM-MO-003-REF-GAP2-HYBRID-ERRORS
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
    - Path: pkg/wsm/git_integration.go
      Note: Diary confirmation that hybrid is default backend
    - Path: pkg/wsm/gitclient/gogit_client.go
      Note: Diary source for ErrNotImplemented sentinel
    - Path: pkg/wsm/gitclient/hybrid_client.go
      Note: Diary evidence for silent error return paths
    - Path: ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.log
      Note: Diary experiment output
    - Path: ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.sh
      Note: Diary experiment automation
ExternalSources: []
Summary: Chronological investigation log for HybridClient error swallowing defect.
LastUpdated: 2026-02-28T14:28:00-05:00
WhatFor: Execution diary and reproducibility notes for intern onboarding.
WhenToUse: Use when retracing evidence and validating the hybrid error propagation fix.
---


# Investigation Diary

## Goal

Produce a reproducible, implementation-ready bug ticket for HybridClient error swallowing (refactor gap 2), with direct proof and a robust fix plan for a new intern.

## Context

Gap 2 from the refactor status doc states that `HybridClient` drops non-fallback errors in mutating methods. Because hybrid is default backend mode, this is not edge behavior.

## Chronological Log

## Step 1: Confirm Hybrid Is Default Path

Command:

```bash
nl -ba workspace-manager/pkg/wsm/git_integration.go | sed -n '1,60p'
```

Finding:

1. `WSM_GIT_BACKEND` defaults to `hybrid` when unset.

Implication:

- The defect is on the default production path.

## Step 2: Inspect Hybrid Method Implementations

Command:

```bash
nl -ba workspace-manager/pkg/wsm/gitclient/hybrid_client.go | sed -n '1,220p'
```

Findings:

1. Several methods follow pattern:
   - if `err == ErrNotImplemented`: fallback
   - otherwise `return nil`
2. This pattern appears in `Add`, `Reset`, `Fetch`, `Push`, `CreateBranch`, `CheckoutBranch`.

Interpretation:

- Non-`ErrNotImplemented` errors are silently discarded.

## Step 3: Design Reproduction Experiment

Created file:

- `scripts/repro_hybrid_error_swallow.sh`

Experiment structure:

1. Build a fake primary backend returning real errors.
2. Build a fake fallback backend with counters.
3. Call `HybridClient.Add` and `HybridClient.Push`.
4. Record returned error and call counters.
5. Run control case with `ErrNotImplemented` to prove fallback path still works.

## Step 4: Execute Experiment and Capture Output

Command:

```bash
cd workspace-manager
./ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.sh
```

Output log:

- `scripts/repro_hybrid_error_swallow.log`

Observed result summary:

1. Case A (real errors): returns `err=<nil>` while fallback call count remains `0`.
2. Case B (`ErrNotImplemented`): fallback call count increments.

Conclusion:

- Silent-error bug is real and isolated to fallback guard logic.

## Step 5: Fix Strategy Selection

Decision:

1. Replace duplicate one-liners with shared fallback helpers.
2. Use `errors.Is(err, ErrNotImplemented)` for sentinel handling.
3. Add table-driven unit tests for all affected methods.

Rationale:

- This addresses immediate bug and removes duplication root cause.

## Quick Reference

### Reproduction Commands

```bash
cd /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager
./ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.sh
sed -n '1,220p' ./ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.log
```

### Expected Pre-fix Output Snippet

```text
Add: err=<nil> primaryCalls=1 fallbackCalls=0
Push: err=<nil> primaryCalls=1 fallbackCalls=0
```

### Expected Post-fix Output Snippet

```text
Add: err=primary add failed primaryCalls=1 fallbackCalls=0
Push: err=primary push failed primaryCalls=1 fallbackCalls=0
```

## Usage Examples

### Example: Intern Verification Loop

```bash
# 1) run baseline repro
bash ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.sh

# 2) implement fix in pkg/wsm/gitclient/hybrid_client.go

# 3) run targeted tests (to be added)
go test ./pkg/wsm/gitclient -run Hybrid -v

# 4) rerun repro and confirm error propagation
bash ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.sh
```

## Related

1. Design doc: `design-doc/01-bug-report-and-fix-plan-hybridclient-error-propagation.md`
2. Parent audit ticket doc: `ttmp/2026/01/09/WSM-MO-001-ANALYZE-REFACTOR--analyze-workspace-manager-refactor/analysis/02-osha-style-code-and-architecture-review.md`

## Implementation Diary (2026-02-28, Phase 2)

### Step A: Refactor HybridClient fallback/error semantics

Files edited:

1. `pkg/wsm/gitclient/hybrid_client.go`
2. `pkg/wsm/gitclient/client.go` (interface expansion)

Key implementation updates:

1. Added helper `shouldFallback(err)` using `errors.Is(err, ErrNotImplemented)`.
2. Replaced direct `== ErrNotImplemented` checks.
3. Fixed mutating methods to return primary errors when real failures occur:
   - `Add`, `Reset`, `Fetch`, `Push`, `CreateBranch`, `CheckoutBranch`
4. Added hybrid support for new branch APIs (`RemoteBranchExists`, later `LocalBranchExists`).

### Step B: Add regression tests

Files added/updated:

1. `pkg/wsm/gitclient/hybrid_client_test.go`

Coverage added:

1. Table-driven tests for mutating methods:
   - real primary errors propagate
   - wrapped `ErrNotImplemented` triggers fallback
2. Regression assertions that fallback is not called on non-fallback errors.
3. Additional fallback/propagation tests for `RemoteBranchExists` path.

### Step C: Re-run reproduction script and fix script drift

Commands:

```bash
./ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.sh
```

First run issue:

- Script-generated fake client no longer satisfied `GitClient` after interface expansion (`missing method RemoteBranchExists`).

Fix:

- Updated script program to implement new interface methods (`RemoteBranchExists`, then `LocalBranchExists`).

Final log outcome:

```text
Case A: primary returns real errors, fallback should not run, error should bubble up
Add: err=primary add failed primaryCalls=1 fallbackCalls=0
Push: err=primary push failed primaryCalls=1 fallbackCalls=0
```

Interpretation:

- Silent-success defect is no longer reproduced.

### Step D: Validation run

Commands:

```bash
go test ./pkg/wsm/gitclient -run 'Hybrid|RemoteBranch' -v
go test ./...
```

Result summary:

1. Hybrid + gitclient targeted tests passed.
2. Full suite still fails in integration scenarios due pre-existing harness issues (stale `.out/wsm` and environment contamination behavior), unrelated to this bug fix logic.

### Step E: Ticket closure and documentation hygiene

Commands:

```bash
docmgr doctor --ticket WSM-MO-003-REF-GAP2-HYBRID-ERRORS --stale-after 30
docmgr ticket close --ticket WSM-MO-003-REF-GAP2-HYBRID-ERRORS --changelog-entry "All gap 2 implementation tasks completed; HybridClient error propagation contract fixed with regression coverage."
```

Result:

- Doctor passed.
- Ticket status moved `active -> complete`.
- Tasks are fully closed.
