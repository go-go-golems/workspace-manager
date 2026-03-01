---
Title: 'Refactor Gap 1: Remote Branch Detection False Negatives'
Ticket: WSM-MO-002-REF-GAP1-REMOTE-BRANCH
Status: complete
Topics:
    - refactor
    - architecture
    - workspace-manager
    - git
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/wsm/gitclient/cli_client.go
      Note: Ticket-wide source for CLI branch semantics
    - Path: pkg/wsm/gitclient/client.go
      Note: Ticket implementation changed GitClient contract
    - Path: pkg/wsm/gitclient/gogit_client.go
      Note: Ticket-wide source for go-git branch semantics
    - Path: pkg/wsm/gitclient/remote_branch_exists_test.go
      Note: Ticket backend regression tests
    - Path: pkg/wsm/workspace.go
      Note: |-
        Ticket-wide source for branch-resolution behavior and bug
        Ticket implementation changed branch resolution behavior
    - Path: pkg/wsm/workspace_branch_test.go
      Note: Ticket workflow regression tests
    - Path: ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.log
      Note: Ticket-wide reproduction evidence
    - Path: ttmp/2026/02/28/WSM-MO-002-REF-GAP1-REMOTE-BRANCH--refactor-gap-1-remote-branch-detection-false-negatives/scripts/repro_remote_branch_false_negative.sh
      Note: Ticket-wide reproduction script
ExternalSources: []
Summary: 'Ticket package for gap 1: diagnosis, reproduction, and fix design for remote branch detection false negatives.'
LastUpdated: 2026-02-28T14:28:32.410864483-05:00
WhatFor: Assignable bug ticket for intern implementation.
WhenToUse: Use when implementing remote-branch existence correctness in worktree flows.
---





# Refactor Gap 1: Remote Branch Detection False Negatives

## Overview

This ticket captures a reproducible bug where `CheckRemoteBranchExists` returns false negatives in all configured backends (`cli`, `gogit`, `hybrid`). The bug affects worktree branch-resolution behavior in workspace creation and add-repo flows.

## Key Links

- [Design Doc](design-doc/01-bug-report-and-fix-plan-remote-branch-detection.md)
- [Investigation Diary](reference/01-investigation-diary.md)
- [Repro Script](scripts/repro_remote_branch_false_negative.sh)
- [Repro Log](scripts/repro_remote_branch_false_negative.log)

## Status

Current status: **active**

## Topics

- refactor
- architecture
- workspace-manager
- git

## Tasks

See [tasks.md](./tasks.md) for implementation checklist.

## Changelog

See [changelog.md](./changelog.md) for evidence and delivery history.

## Structure

- design-doc/ - Primary bug-report and fix design
- reference/ - Chronological investigation diary
- scripts/ - Reproduction scripts and logs
