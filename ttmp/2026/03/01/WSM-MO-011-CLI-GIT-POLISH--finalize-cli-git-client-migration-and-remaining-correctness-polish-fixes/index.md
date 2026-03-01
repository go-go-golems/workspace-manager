---
Title: Finalize CLI Git Client Migration and Remaining Correctness/Polish Fixes
Ticket: WSM-MO-011-CLI-GIT-POLISH
Status: complete
Topics:
    - workspace-manager
    - git
    - cli
    - testing
    - js-api
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: Makefile
      Note: CLI-only Dagger targets
    - Path: ci/dagger/main.go
      Note: CLI-only Dagger pipeline
    - Path: pkg/wsm/git_utils.go
      Note: Merged and needs-rebase semantics currently tied to default branch assumptions
    - Path: pkg/wsm/gitclient/cli_client.go
      Note: CLI status parsing and commit contract behavior
    - Path: pkg/wsm/gitclient/status_worktree_cli_test.go
      Note: Unit coverage for parser edge cases
    - Path: pkg/wsm/gitclient/worktree_cli.go
      Note: |-
        Worktree list parsing should use machine-friendly format
        Porcelain-based worktree parsing
    - Path: pkg/wsmjs/spec/wsm.d.ts
      Note: JS API contract reference for manager and git namespaces
    - Path: test/integration/scenarios/status_diff_test.go
      Note: Integration coverage currently checks shape but not merged/rebase semantics
    - Path: test/integration/scenarios/status_semantics_test.go
      Note: Integration regression coverage for merged/rebase fields
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-01T11:24:44.792862008-05:00
WhatFor: ""
WhenToUse: ""
---




# Finalize CLI Git Client Migration and Remaining Correctness/Polish Fixes

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
- cli
- testing
- js-api

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
