---
Title: Unify WSM Commands Around Glazed Dual Output Pattern
Ticket: WSM-MO-009-GLAZED-DUAL-OUTPUT-WSM
Status: active
Topics:
    - architecture
    - glazed
    - refactor
    - workspace-manager
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Implementation planning ticket for migrating WSM commands to native Glazed dual-mode output with grouped-verb directory normalization."
LastUpdated: 2026-03-01T05:51:48.998770015-05:00
WhatFor: "Track planning artifacts for unifying WSM command output around Run/RunIntoGlazeProcessor boundaries and grouped command layout."
WhenToUse: "Use this ticket when implementing or reviewing WSM Glazed dual-mode migration work."
---

# Unify WSM Commands Around Glazed Dual Output Pattern

## Overview

This ticket captures the implementation plan for moving WSM commands from custom `output-mode` + `EmitRows` helpers to native Glazed dual-mode (`Run` for human output, `RunIntoGlazeProcessor` for structured rows).

It also includes a structural layout decision: grouped verbs should be split into mirrored subdirectories with their own `root.go` registrars (for example `git/branch/*`, `git/rebase/*`, `registry/list/*`).

## Key Links

- Design doc: `design-doc/01-wsm-glazed-dual-output-implementation-plan.md`
- Investigation diary: `reference/01-investigation-diary.md`
- Related files and sources: see frontmatter `RelatedFiles` and `ExternalSources`

## Status

Current status: **active**

## Topics

- architecture
- glazed
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
