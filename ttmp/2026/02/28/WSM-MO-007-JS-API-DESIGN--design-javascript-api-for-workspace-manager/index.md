---
Title: Design JavaScript API for Workspace Manager
Ticket: WSM-MO-007-JS-API-DESIGN
Status: active
Topics:
    - architecture
    - api-design
    - workspace-manager
    - javascript
    - goja
    - geppetto
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket workspace for designing a JavaScript API over workspace-manager using evidence from go-go-goja and geppetto JS module patterns.
LastUpdated: 2026-02-28T19:42:00-05:00
WhatFor: Track and deliver the JS API architecture decision document for user review.
WhenToUse: Use when implementing or reviewing `require("wsm")` API design choices.
---

# Design JavaScript API for Workspace Manager

## Overview

This ticket evaluates and proposes a JavaScript API for Workspace Manager functionality. The analysis is grounded in:

1. Current `workspace-manager` package/workflow architecture.
2. Native module/runtime patterns from `go-go-goja`.
3. JS API ergonomics and typing patterns from `geppetto/js`.

Primary deliverables in this ticket:

1. `design-doc/01-workspace-manager-javascript-api-brainstorm-and-design-options.md`
2. `reference/01-investigation-diary.md`

## Key Links

- **Design Doc**: `design-doc/01-workspace-manager-javascript-api-brainstorm-and-design-options.md`
- **Diary**: `reference/01-investigation-diary.md`
- **Tasks**: [tasks.md](./tasks.md)
- **Changelog**: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- api-design
- workspace-manager
- javascript
- goja
- geppetto

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
