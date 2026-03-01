---
Title: Remaining CLI Git Migration Issues and Fix Plan
Ticket: WSM-MO-011-CLI-GIT-POLISH
Status: active
Topics:
    - workspace-manager
    - git
    - cli
    - testing
    - js-api
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Track remaining hardcoded assumptions, parser robustness, contract cleanup, and test gaps after CLI-only backend migration.
LastUpdated: 2026-03-01T11:11:42.71818659-05:00
WhatFor: ""
WhenToUse: ""
---

# Remaining CLI Git Migration Issues and Fix Plan

## Executive Summary

The CLI-only backend migration is in place, but a few correctness and maintainability gaps remain. This ticket scopes targeted fixes for branch-default assumptions, parser robustness, commit contract clarity, and missing tests that allowed a behavior regression to pass.

## Problem Statement

Current behavior still includes a mix of hardcoded assumptions and weak parsing paths:

- Merged/rebase status checks are tied to `origin/main` semantics and can misreport in non-`main` repositories.
- `git status` and `git worktree list` parsing rely on human-oriented output formats.
- `GitClient.Commit` returns a string but currently does not return a meaningful commit ID.
- Tests validate command availability and broad parity, but do not assert key semantic fields (for example `is_merged` and `needs_rebase`).
- Some migration leftovers still reference removed `hybrid/gogit` backend language in CI-oriented files.

## Proposed Solution

Address the remaining gaps in small, reviewable phases:

- Add configurable base branch resolution with default `main` fallback, and route status semantics through it.
- Replace brittle parsers with machine-safe output parsing (`--porcelain -z` and `git worktree list --porcelain`).
- Resolve the commit contract by either returning the new commit hash or simplifying/removing the unused return value in the interface.
- Add integration and unit tests that assert semantic status behavior and parser edge cases.
- Remove stale backend matrix references that imply `hybrid/gogit` support.

## Design Decisions

- Keep `main` as the default base branch to preserve existing behavior, but make it configurable to support repositories with different defaults.
- Prefer deterministic machine formats over human output parsing whenever available.
- Treat semantic status fields as contract-level behavior and require explicit tests for them.
- Keep scope bounded to migration completion and correctness, not broad feature expansion.

## Alternatives Considered

- Keep hardcoded `origin/main` and document as limitation.
Rejected because it causes incorrect behavior for repositories using different default branches.

- Leave parser logic as-is and only patch edge cases.
Rejected because ad-hoc parsing remains fragile and expensive to maintain.

- Add more smoke tests only.
Rejected because smoke tests did not catch the semantic regression; field-level assertions are required.

## Implementation Plan

1. Implement configurable base branch handling with `main` default.
2. Refactor git status and worktree parsing to machine-safe formats.
3. Fix or simplify `GitClient.Commit` return contract.
4. Add semantic and parser-focused tests.
5. Clean migration leftovers in CI/Makefile wording and options.
6. Validate with full `go test ./...` and representative workspace status comparisons.

## Open Questions

- Configuration shape for base branch selection.
- Whether the commit hash return is needed by any downstream caller or can be removed cleanly.
- Whether any external scripts still rely on legacy backend matrix wording.

## References

- Prior migration ticket: `WSM-MO-010-CLI-ONLY-GIT-BACKEND`
