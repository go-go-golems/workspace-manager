---
Title: Repo-Wide Branch Abstraction Refactor (No Backward Compatibility)
Ticket: WSM-MO-004-BRANCH-ABSTRACTION-REPO-WIDE
Status: complete
Topics:
    - architecture
    - refactor
    - workspace-manager
    - git
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/wsm/discovery.go
      Note: Branch discovery call-site migration target
    - Path: pkg/wsm/git_utils.go
      Note: Legacy branch utility migration target
    - Path: pkg/wsm/gitclient/client.go
      Note: Current GitClient contract target for breaking branch API redesign
    - Path: pkg/wsm/workspace.go
      Note: Current workspace branch decision logic migration target
ExternalSources: []
Summary: Repo-wide breaking refactor plan to introduce a clean branch abstraction and remove ambiguous branch semantics.
LastUpdated: 2026-02-28T15:04:59.121334992-05:00
WhatFor: Execution anchor ticket for full branch abstraction redesign.
WhenToUse: Use for implementing and tracking the no-backwards-compatibility branch refactor.
---



# Repo-Wide Branch Abstraction Refactor (No Backward Compatibility)

## Overview

This ticket defines a full breaking-change migration to a clean branch abstraction layer (`BranchService`) so all branch decisions are explicit, typed, and centralized.

## Key Links

- [Design Doc](design-doc/01-implementation-plan-repo-wide-branch-abstraction.md)
- [Implementation Diary](reference/01-implementation-diary.md)
- [Tasks](tasks.md)

## Status

Current status: **complete**

## Topics

- architecture
- refactor
- workspace-manager
- git

## Tasks

See [tasks.md](./tasks.md) for the detailed phased checklist.

## Changelog

See [changelog.md](./changelog.md) for updates.
