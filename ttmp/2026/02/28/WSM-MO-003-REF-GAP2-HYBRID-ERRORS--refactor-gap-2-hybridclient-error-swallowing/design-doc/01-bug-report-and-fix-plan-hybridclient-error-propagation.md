---
Title: 'Bug Report and Fix Plan: HybridClient Error Propagation'
Ticket: WSM-MO-003-REF-GAP2-HYBRID-ERRORS
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
    - Path: pkg/wsm/git_integration.go
      Note: Hybrid backend default selection
    - Path: pkg/wsm/gitclient/client.go
      Note: Error contract surface for mutating operations
    - Path: pkg/wsm/gitclient/gogit_client.go
      Note: ErrNotImplemented sentinel source
    - Path: pkg/wsm/gitclient/hybrid_client.go
      Note: Core bug location and fallback logic target
    - Path: ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.log
      Note: Captured failing run evidence
    - Path: ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.sh
      Note: Deterministic bug reproduction script
ExternalSources: []
Summary: Detailed bug report and implementation plan for HybridClient methods that currently swallow non-fallback errors.
LastUpdated: 2026-02-28T14:24:00-05:00
WhatFor: Critical correctness fix planning and onboarding artifact for intern implementation.
WhenToUse: Use when touching fallback semantics and error contracts in the git backend layer.
---


# Bug Report and Fix Plan: HybridClient Error Propagation

## Executive Summary

This ticket documents refactor gap 2: multiple mutating methods in `HybridClient` return `nil` even when the primary backend returns a real error. That behavior makes failed git operations look successful.

Affected methods today:

1. `Add`
2. `Reset`
3. `Fetch`
4. `Push`
5. `CreateBranch`
6. `CheckoutBranch`

The pattern is copy-pasted in one-line implementations and is inconsistent with the intended fallback contract described in the file comment.

This is a critical correctness bug. False success can corrupt user trust, break automation, and cause hidden divergence between workspace state and reported operation status.

## Bug Report

### ID and Severity

- Ticket ID: `WSM-MO-003-REF-GAP2-HYBRID-ERRORS`
- Severity: `Critical`
- Category: error-contract violation / hidden operation failure
- Affected backend mode: default `hybrid` mode (`WSM_GIT_BACKEND` unset)

### Expected Behavior

For every method in `HybridClient`:

1. If primary returns `nil`, return `nil`.
2. If primary returns `ErrNotImplemented`, call fallback and return fallback result.
3. If primary returns any other error, return that error unchanged (or wrapped predictably).

### Actual Behavior

For the six methods listed above:

1. primary returns real error
2. method returns `nil`
3. fallback is not called

### Evidence Anchors

1. `HybridClient` file-level contract:
   - `pkg/wsm/gitclient/hybrid_client.go:5-7`
2. Broken methods returning `nil` after non-`ErrNotImplemented` errors:
   - `pkg/wsm/gitclient/hybrid_client.go:50-57`
   - `pkg/wsm/gitclient/hybrid_client.go:68-75`
   - `pkg/wsm/gitclient/hybrid_client.go:81-88`
3. Hybrid backend is default in integration factory:
   - `pkg/wsm/git_integration.go:10-23`

### Reproduction Evidence

Script:

- `scripts/repro_hybrid_error_swallow.sh`

Observed output (`scripts/repro_hybrid_error_swallow.log`):

```text
Case A: primary returns real errors, fallback should not run, error should bubble up
Add: err=<nil> primaryCalls=1 fallbackCalls=0
Push: err=<nil> primaryCalls=1 fallbackCalls=0
```

This proves the silent-error bug.

For control case (`ErrNotImplemented`), fallback runs as expected.

## Why This Happens (Root Cause)

### Root Cause 1: Copy-Pasted One-Liners With Incorrect Return Logic

Broken pattern:

```go
if err := h.primary.Add(...); err == ErrNotImplemented { return h.fallback.Add(...) }
return nil
```

This pattern ignores any error value that is not exactly `ErrNotImplemented`.

### Root Cause 2: No Shared Helper for Fallback Semantics

Fallback logic is manually duplicated across many methods in `HybridClient`. Duplication increases probability of subtle divergence, as already happened.

### Root Cause 3: Missing Unit Tests for Contract Invariants

There are no direct unit tests in `pkg/wsm/gitclient` proving that real errors are propagated and only `ErrNotImplemented` triggers fallback.

## Concepts the Intern Must Understand

### 1) Error Contract Invariant

Fallback wrappers are policy layers. They do not reinterpret successful vs failed operations. Returning `nil` when operation failed violates the contract and hides state loss.

### 2) Sentinel Errors and `errors.Is`

Code currently compares with `==`. Safer implementation uses `errors.Is(err, ErrNotImplemented)` so wrapped sentinel values still trigger fallback.

### 3) Why This Is Operationally Dangerous

These are mutating operations. If `Push` fails but returns success, CI/CD or operators may assume remote state is updated when it is not. If `Add` fails but returns success, later commit steps fail unexpectedly with confusing diagnostics.

## Scope

### In Scope for This Ticket

1. Fix error propagation in all affected `HybridClient` methods.
2. Normalize fallback check to `errors.Is(..., ErrNotImplemented)`.
3. Add unit tests for fallback and propagation behavior per method.
4. Reduce duplication in hybrid fallback implementation.

### Out of Scope

1. Full redesign of all git abstractions.
2. Replacing hybrid with a different architecture.
3. Non-hybrid backend behavior changes beyond compile/test necessities.

## Proposed Fix Design

## Decision

Refactor `HybridClient` methods to enforce one shared fallback rule and remove copy-pasted method-specific error handling.

### Core Helper Pattern

Add internal helper functions in `hybrid_client.go`:

```go
func (h *HybridClient) fallbackErr(primaryErr error, fallback func() error) error {
    if primaryErr == nil {
        return nil
    }
    if errors.Is(primaryErr, ErrNotImplemented) {
        return fallback()
    }
    return primaryErr
}
```

```go
func fallbackValue[T any](value T, err error, fallback func() (T, error)) (T, error) {
    if err == nil {
        return value, nil
    }
    if errors.Is(err, ErrNotImplemented) {
        return fallback()
    }
    return value, err
}
```

Use this consistently in all methods.

### Method-by-Method Fix Expectations

1. `Add`: return primary error when real failure; no silent `nil`.
2. `Reset`: same.
3. `Fetch`: same.
4. `Push`: same.
5. `CreateBranch`: same.
6. `CheckoutBranch`: same.

Also standardize existing read/value methods to `errors.Is` for future-proofing wrapped sentinel errors.

### Example Correct Implementation

```go
func (h *HybridClient) Add(ctx context.Context, repo RepositoryHandle, path string) error {
    err := h.primary.Add(ctx, repo, path)
    return h.fallbackErr(err, func() error {
        return h.fallback.Add(ctx, repo, path)
    })
}
```

## Alternatives Considered

### Alternative A: Minimal Line Edit Only on Broken Methods

Fix six methods in place, keep duplication.

Rejected because:

1. high chance of future regressions
2. inconsistent semantics likely to return
3. misses opportunity to centralize contract policy

### Alternative B: Keep Equality (`==`) Comparison

Rejected because wrapped `ErrNotImplemented` would not match and fallback could fail unexpectedly.

### Alternative C: Make Hybrid Always Try Fallback on Any Error

Rejected because it would hide real errors and could lead to unsafe mixed behavior.

## Detailed Implementation Plan

### Phase 1: Refactor HybridClient Internals

1. Add helper(s) for fallback decision and error propagation.
2. Replace current one-liners with explicit error-first logic.
3. Update all methods (not only six) to use the same fallback decision primitive.

### Phase 2: Add Unit Tests

Create `pkg/wsm/gitclient/hybrid_client_test.go` with table-driven tests covering:

1. primary success
2. primary `ErrNotImplemented` -> fallback used
3. primary real error -> returned unchanged

Target methods:

1. `Add`
2. `Reset`
3. `Fetch`
4. `Push`
5. `CreateBranch`
6. `CheckoutBranch`
7. one value-returning method (`ListBranches` or `Status`) to validate helper consistency

### Phase 3: Verification

1. run `go test ./pkg/wsm/gitclient -run Hybrid -v`
2. run broader `go test ./...` (or integration subset if full suite not hermetic)
3. rerun `scripts/repro_hybrid_error_swallow.sh`

Expected post-fix script result for Case A:

```text
Add: err=primary add failed ...
Push: err=primary push failed ...
```

## Test Design (Concrete)

### Test Fixture Design

Use lightweight fake clients with per-method configurable errors and call counters.

Assertions:

1. Fallback call count stays `0` for primary real error.
2. Fallback call count becomes `1` for `ErrNotImplemented`.
3. Returned error matches expected source.

### Regression Guard

Add explicit regression test names, for example:

1. `TestHybridAdd_PropagatesPrimaryError`
2. `TestHybridPush_PropagatesPrimaryError`

This keeps failure signal obvious if bug reappears.

## Risks and Mitigations

1. Risk: behavior change may surface hidden failures in existing workflows.
   - Mitigation: this is desired; document as correctness fix in changelog/PR.
2. Risk: helper abstraction could obscure method-specific nuances.
   - Mitigation: keep helper small and single-purpose; maintain direct unit tests per method.
3. Risk: test fragility if fake clients incomplete.
   - Mitigation: keep fake minimal but interface-complete and deterministic.

## Acceptance Criteria

1. No hybrid mutating method returns `nil` when primary returns real error.
2. Fallback only triggers for `ErrNotImplemented` (including wrapped sentinel via `errors.Is`).
3. New unit tests cover all affected methods and pass.
4. Reproduction script output proves error propagation is fixed.

## Intern Runbook

1. Run reproduction script and capture baseline output.
2. Implement helper + method refactor in `hybrid_client.go`.
3. Add unit tests with fake clients.
4. Run target tests, then broader tests.
5. Re-run reproduction and attach before/after logs.

## References

1. `pkg/wsm/gitclient/hybrid_client.go:5-88`
2. `pkg/wsm/gitclient/gogit_client.go:14`
3. `pkg/wsm/git_integration.go:10-23`
4. `pkg/wsm/gitclient/client.go:47-74`
5. `scripts/repro_hybrid_error_swallow.sh`
6. `scripts/repro_hybrid_error_swallow.log`
