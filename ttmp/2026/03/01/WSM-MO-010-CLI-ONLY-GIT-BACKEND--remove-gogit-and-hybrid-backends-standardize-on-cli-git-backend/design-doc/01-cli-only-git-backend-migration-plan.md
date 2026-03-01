---
Title: CLI-only git backend migration plan
Ticket: WSM-MO-010-CLI-ONLY-GIT-BACKEND
Status: active
Topics:
    - architecture
    - git
    - refactor
    - workspace-manager
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-01T10:45:46.991915986-05:00
WhatFor: "Define and execute a breaking simplification to remove gogit/hybrid backends and standardize WSM on git CLI only."
WhenToUse: "Use when implementing or reviewing backend simplification and related test/docs cleanup."
---

# CLI-only git backend migration plan

## Executive Summary

WSM currently carries three backend modes (`cli`, `gogit`, `hybrid`) while most production-relevant behavior already relies on CLI semantics, particularly around worktrees and unsupported operations in go-git.

This design removes backend selection and standardizes all git operations on the CLI backend with no backward-compatibility shims.

## Problem Statement

- Hybrid/gogit status behavior can diverge from CLI behavior in real worktree-heavy repos.
- Backend matrix support increases maintenance and test surface without delivering consistent value.
- Environment-driven backend selection (`WSM_GIT_BACKEND`) creates ambiguity in debugging and support.

## Proposed Solution

Implement a CLI-only backend architecture:

- `BuildGitBackends` always returns `gitclient.NewCli()` + `gitclient.NewCliWorktrees()`.
- Remove `WSM_GIT_BACKEND` selection logic.
- Delete `pkg/wsm/gitclient/gogit_client.go` and `pkg/wsm/gitclient/hybrid_client.go` and their dedicated tests.
- Update tests and helpers to stop setting or matrix-testing backend variants.
- Update architecture docs to reflect CLI-only backend.
- Clean module dependencies to remove go-git.

## Design Decisions

- No compatibility mode: backend env selection is removed entirely, not deprecated.
- Keep existing `GitClient` interface for now to minimize ripple effects.
- Preserve behavior by standardizing on current CLI implementation semantics.
- Delete backend-specific tests instead of keeping dead abstractions.

## Alternatives Considered

- Keep hybrid default but force CLI for `Status`: rejected; still leaves split backend complexity.
- Keep gogit as opt-in env mode: rejected per ticket requirement ("no backwards compatibility").
- Fix gogit status classification only: rejected as partial mitigation, not simplification.

## Implementation Plan

1. Update backend construction to CLI-only.
2. Remove gogit/hybrid implementations and backend selection references.
3. Update backend matrix tests and integration helpers/scenarios.
4. Update docs and module dependencies.
5. Run full lint/tests and commit in phases with diary/changelog evidence.

## Open Questions

- None; scope is explicit and breaking by intent.

## References

- `pkg/wsm/git_integration.go`
- `pkg/wsm/gitclient/cli_client.go`
- `pkg/wsm/gitclient/gogit_client.go`
- `pkg/wsm/gitclient/hybrid_client.go`
