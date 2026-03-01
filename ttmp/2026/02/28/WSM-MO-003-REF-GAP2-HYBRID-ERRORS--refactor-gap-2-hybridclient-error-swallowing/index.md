---
Title: 'Refactor Gap 2: HybridClient Error Swallowing'
Ticket: WSM-MO-003-REF-GAP2-HYBRID-ERRORS
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
    - Path: pkg/wsm/git_integration.go
      Note: Ticket-wide source for hybrid default behavior
    - Path: pkg/wsm/gitclient/gogit_client.go
      Note: Ticket-wide source for ErrNotImplemented sentinel
    - Path: pkg/wsm/gitclient/hybrid_client.go
      Note: |-
        Ticket-wide source for silent error bug
        Ticket implementation fixed fallback propagation logic
    - Path: pkg/wsm/gitclient/hybrid_client_test.go
      Note: Ticket regression tests for fallback semantics
    - Path: ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.log
      Note: Ticket-wide reproduction evidence
    - Path: ttmp/2026/02/28/WSM-MO-003-REF-GAP2-HYBRID-ERRORS--refactor-gap-2-hybridclient-error-swallowing/scripts/repro_hybrid_error_swallow.sh
      Note: Ticket-wide reproduction script
ExternalSources: []
Summary: 'Ticket package for gap 2: diagnosis, reproduction, and fix design for HybridClient error swallowing.'
LastUpdated: 2026-02-28T14:28:32.423626663-05:00
WhatFor: Assignable critical bug ticket for intern implementation.
WhenToUse: Use when implementing fallback error semantics in HybridClient.
---





# Refactor Gap 2: HybridClient Error Swallowing

## Overview

This ticket captures a critical bug in default hybrid backend mode where mutating git operations can fail but still return success (`nil` error). The issue is reproduced with a targeted script and explained with file-level evidence.

## Key Links

- [Design Doc](design-doc/01-bug-report-and-fix-plan-hybridclient-error-propagation.md)
- [Investigation Diary](reference/01-investigation-diary.md)
- [Repro Script](scripts/repro_hybrid_error_swallow.sh)
- [Repro Log](scripts/repro_hybrid_error_swallow.log)

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
