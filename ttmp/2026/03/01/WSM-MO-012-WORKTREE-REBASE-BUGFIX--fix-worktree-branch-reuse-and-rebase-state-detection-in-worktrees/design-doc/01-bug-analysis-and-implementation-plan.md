---
Title: Bug Analysis and Implementation Plan
Ticket: WSM-MO-012-WORKTREE-REBASE-BUGFIX
Status: active
Topics:
    - workspace-manager
    - git
    - worktree
    - rebase
    - bugfix
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Analyze and plan fixes for worktree branch reuse and rebase-state detection in git worktrees.
LastUpdated: 2026-03-01T11:32:09.970449187-05:00
WhatFor: ""
WhenToUse: ""
---

# Bug Analysis and Implementation Plan

## Executive Summary

Two regressions remain in the CLI-only git backend:

1. Worktree creation for existing local branches currently forces `-b`, which attempts to create a new branch and fails when that branch already exists.
2. Rebase in-progress detection uses `repoPath/.git/rebase-*` paths directly, which is incorrect for worktrees where `.git` is a pointer file.

This ticket fixes both issues with targeted code changes and regression tests.

## Problem Statement

### Bug A: Existing branch reuse fails during worktree add

- Current implementation always adds `-b <branch>` when `branch != ""` in `gitclient.CliWorktrees.Add`.
- Callers in `WorkspaceManager` use this path for `ResolutionStrategyUseLocal`, where the branch already exists and should be reused.
- Result: `git worktree add -b <existing-branch> ...` fails with "branch already exists" in common flows.

### Bug B: Rebase in-progress state not detected reliably in worktrees

- Rebase status checks look for:
  - `<repoPath>/.git/rebase-merge`
  - `<repoPath>/.git/rebase-apply`
- In git worktrees, `.git` is a file that points at the real gitdir under `.git/worktrees/...`.
- Result: rebase can be reported as `none` even when in progress (notably after conflicts are resolved but before `rebase --continue`).

## Proposed Solution

### Fix A (worktree branch reuse)

- Extend `WorktreeAddOptions` with explicit branch mode for reuse semantics.
- For "use existing local branch" flows, call `git worktree add <path> <branch>` (no `-b`).
- Keep existing `-b` behavior for branch creation flows and `-B` behavior for overwrite flows.

### Fix B (rebase state in worktrees)

- Resolve rebase marker paths from actual gitdir via git itself:
  - `git rev-parse --git-path rebase-merge`
  - `git rev-parse --git-path rebase-apply`
- Use those resolved paths to determine in-progress state instead of manual `.git` path joining.

## Design Decisions

- Add explicit worktree add intent via options instead of inferring from string combinations.
- Prefer `git rev-parse --git-path` over parsing `.git` indirection manually.
- Add regression tests close to the failing behavior:
  - gitclient unit test for branch reuse
  - rebase status test for worktree in-progress state

## Alternatives Considered

- Continue using `-b` and special-case branch names.
Rejected because it is semantically wrong for existing branch checkout.

- Parse `.git` file manually to find gitdir.
Rejected because `git rev-parse --git-path` is simpler and more robust across layouts.

## Implementation Plan

1. Fix worktree add behavior for existing local branch reuse.
2. Add/adjust tests for worktree branch-reuse semantics.
3. Fix rebase in-progress detection using git-resolved gitdir paths.
4. Add tests covering in-progress rebase detection in worktree layout.
5. Run targeted and full validation, update ticket docs, close ticket.

## Open Questions

- None currently; behavior and acceptance criteria are clear.

## References

- `pkg/wsm/gitclient/worktree_cli.go`
- `pkg/wsm/workspace.go`
- `pkg/wsm/rebase_operations.go`
