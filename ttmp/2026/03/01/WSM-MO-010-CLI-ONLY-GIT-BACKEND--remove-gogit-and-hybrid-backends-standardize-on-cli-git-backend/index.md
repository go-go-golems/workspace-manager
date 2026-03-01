---
Title: Remove gogit and hybrid backends; standardize on CLI git backend
Ticket: WSM-MO-010-CLI-ONLY-GIT-BACKEND
Status: complete
Topics:
    - architecture
    - git
    - refactor
    - workspace-manager
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Execution ticket for removing gogit/hybrid support and standardizing WSM on CLI git backend."
LastUpdated: 2026-03-01T10:52:24-05:00
WhatFor: "Track implementation, validation, and migration evidence for CLI-only git backend standardization."
WhenToUse: "Use when reviewing or auditing the backend simplification change set."
---

# Remove gogit and hybrid backends; standardize on CLI git backend

## Overview

This ticket delivered a breaking backend simplification: `wsm` now uses the CLI git backend exclusively. `gogit` and `hybrid` implementations, related compatibility selectors, and backend matrix tests were removed.

## Key Links

- Design doc: `design-doc/01-cli-only-git-backend-migration-plan.md`
- Diary: `reference/01-investigation-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **complete**

## Topics

- architecture
- git
- refactor
- workspace-manager

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
