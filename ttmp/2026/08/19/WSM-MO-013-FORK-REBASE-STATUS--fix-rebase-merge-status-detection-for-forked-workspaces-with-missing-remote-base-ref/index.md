---
Title: Fix rebase/merge status detection for forked workspaces with missing remote base ref
Ticket: WSM-MO-013-FORK-REBASE-STATUS
Status: active
Topics:
    - workspace-manager
    - git
    - rebase
    - fork
    - bugfix
    - status
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/wsm/branch/resolver.go
      Note: resolveFromState is the existing resolution policy to mirror for ResolveBaseRef
    - Path: repo://pkg/wsm/branch/types.go
      Note: RemoteTrackingRef/ResolveBaseBranch build the origin/<base> string used by the buggy checks
    - Path: repo://pkg/wsm/git_utils.go
      Note: CheckBranchMerged/CheckBranchNeedsRebase hardcode origin/<base> and swallow/return errors on missing ref
    - Path: repo://pkg/wsm/gitclient/cli_client.go
      Note: RemoteTrackingBranchExists and LocalBranchExists are the for-each-ref primitives to reuse
    - Path: repo://pkg/wsm/status.go
      Note: getRepositoryStatusWithClient calls the two checks with err==nil guards (lines 151-157)
    - Path: repo://pkg/wsm/types.go
      Note: RepositoryStatus IsMerged/NeedsRebase bool fields and comments encoding the buggy assumption
    - Path: repo://pkg/wsm/workflows/fork_workflow.go
      Note: Plan detects baseBranch from source current branch (lines 71-88), causing local-only base on fork
ExternalSources: []
Summary: ""
LastUpdated: 2026-08-19T10:24:25.130856066-04:00
WhatFor: ""
WhenToUse: ""
---








# Fix rebase/merge status detection for forked workspaces with missing remote base ref

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
- rebase
- fork
- bugfix
- status

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
