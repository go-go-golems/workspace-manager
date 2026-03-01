---
Title: Fix worktree branch reuse and rebase-state detection in worktrees
Ticket: WSM-MO-012-WORKTREE-REBASE-BUGFIX
Status: complete
Topics:
    - workspace-manager
    - git
    - worktree
    - rebase
    - bugfix
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/wsm/gitclient/worktree_cli.go
      Note: Worktree add behavior for existing vs new branches
    - Path: pkg/wsm/rebase_operations.go
      Note: Rebase in-progress detection in worktree context
    - Path: pkg/wsm/rebase_operations_test.go
      Note: Tests that reproduce and lock rebase-state behavior in worktrees
    - Path: pkg/wsm/workflows/rebase_workflow.go
      Note: Workflow conflict signaling now reuses shared rebase-state detection
    - Path: pkg/wsm/workspace.go
      Note: Callers selecting ResolutionStrategyUseLocal
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-01T11:41:11.320301576-05:00
WhatFor: ""
WhenToUse: ""
---





# Fix worktree branch reuse and rebase-state detection in worktrees

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- workspace-manager
- git
- worktree
- rebase
- bugfix
- testing

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
