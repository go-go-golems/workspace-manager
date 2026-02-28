---
Title: Investigation diary
Ticket: WSM-MO-005-CMD-PKG-REARCH-REMOVE-SYNC
Status: active
Topics:
    - architecture
    - refactor
    - workspace-manager
    - git
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/root.go
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_sync.go
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/sync_operations.go
ExternalSources: []
Summary: "Chronological investigation log for sync removal planning and command-to-pkg architecture cleanup design."
LastUpdated: 2026-02-28T17:03:00-05:00
WhatFor: "Provide continuation-ready evidence and rationale for WSM-MO-005."
WhenToUse: "Use when implementing the sync removal and command extraction plan."
---

# Diary

## Goal

Create a new ticket and produce an exhaustive implementation-ready architecture plan to remove `sync` and move command-embedded logic into reusable `pkg/` APIs.

## Step 1: Ticket Bootstrap and Scope Anchoring

I started by creating a new docmgr ticket dedicated to this effort so the work has a clean lifecycle, clear scope, and task checklist separate from WSM-MO-004. I created both a design document and an investigation diary document immediately so architecture conclusions and process evidence would stay synchronized.

This step established the execution container and ensured the requested scope included both product-level removal of `sync` and structural cleanup of command-layer logic.

### Prompt Context

**User prompt (verbatim):** "remove sync altogether. 

What we are going to do is create a new docmgr ticket, and add task to remove sync, but also go over all the verbs we have, classify them by complexity, make a list of the funcitonality that is in the cmd_ file itself, how it could be cleaned and moved or merged with what we have in pkg/, including how pkg/ stuff might evolve, the gfoal is to have all the actual funcitonality as a nice reusable package in pkg/ 

Write a detailed design and analysis document in a very verbose style, with prose paragraphs and bullet points and pseudocode and code snippets and API reference and diagrams, for our intern that doesn't know anything aobut the codebase. 

Upload to remarkable."

**Assistant interpretation:** Open a new architecture/research ticket and produce a deep intern-oriented design document covering sync removal and broad CLI-to-pkg extraction strategy, with explicit verb complexity analysis and migration tasks.

**Inferred user intent:** Simplify product surface and improve architecture maintainability by making command code thin and `pkg/` reusable.

### What I did
- Ran `docmgr status --summary-only`.
- Created ticket: `WSM-MO-005-CMD-PKG-REARCH-REMOVE-SYNC`.
- Created docs:
  - `design-doc/01-command-verb-complexity-audit-and-pkg-extraction-plan.md`
  - `reference/01-investigation-diary.md`

### Why
- The requested work is cross-cutting and needs explicit ticket-level tracking.
- A dedicated ticket avoids contaminating prior gap/refactor tickets.

### What worked
- Ticket and docs were created successfully in `ttmp/2026/02/28/...`.

### What didn't work
- N/A.

### What I learned
- Creating docs at ticket start reduces context drift while doing heavy architecture audits.

### What was tricky to build
- Choosing a ticket identifier that captures both sync removal and command/package architecture without becoming ambiguous.

### What warrants a second pair of eyes
- Ticket naming consistency (`REARCH` spelling in ID) if strict naming conventions are enforced.

### What should be done in the future
- Consider standardized ticket slug linting to prevent minor naming typos.

### Code review instructions
- Verify ticket path and files exist.
- Confirm `tasks.md` includes explicit sync-removal tasks.

### Technical details
- Ticket path: `ttmp/2026/02/28/WSM-MO-005-CMD-PKG-REARCH-REMOVE-SYNC--remove-sync-and-refactor-cli-verbs-into-reusable-pkg-api`.

## Step 2: Evidence Collection and Verb Complexity Audit

I performed a code-level survey to classify every root verb by complexity and collect objective evidence about where business logic currently lives. This was intentionally evidence-first to avoid speculative architecture recommendations.

I used a mix of command inventory, line counts, helper-function discovery, and line-anchored file reads for high-complexity command paths.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Build an evidence-backed map of command complexity and command-to-pkg boundaries before drafting design recommendations.

**Inferred user intent:** Make refactor decisions auditable for a new intern.

### What I did
- Collected root command registration from `cmd/wsm/root.go`.
- Enumerated all command files in `cmd/cmds`.
- Generated LoC table: `wc -l cmd/cmds/*.go`.
- Collected complexity signals per file:
  - direct shell execution count,
  - `wsm.*` call count,
  - concurrency usage (`errgroup`/`semaphore`),
  - interactive prompting signals.
- Collected line-anchored excerpts from:
  - `cmd_sync.go`, `cmd_branch.go`, `cmd_diff.go`, `cmd_status.go`,
  - `cmd_pr.go`, `cmd_push.go`, `cmd_rebase.go`, `cmd_merge.go`, `cmd_tmux.go`,
  - `pkg/wsm/sync_operations.go`, `pkg/wsm/git_operations.go`, `pkg/wsm/status.go`.

### Why
- Verb complexity classification should come from concrete implementation characteristics, not guesswork.
- Extraction recommendations must be tied to specific code hotspots.

### What worked
- The metrics quickly identified high-risk commands (`rebase`, `merge`, `pr`, `push`, `status`) and coupling around `sync_operations.go`.
- Evidence showed `SyncOperations` currently contains non-sync concerns used by `branch` and `log` command paths.

### What didn't work
- N/A.

### What I learned
- The largest architectural issue is not only `sync` command existence; it is conceptual overloading of `SyncOperations` with branch and history behavior.

### What was tricky to build
- Distinguishing complexity from mere file size required combining multiple signals (side effects, external tool usage, concurrency, interactive flow).

### What warrants a second pair of eyes
- Complexity thresholds: whether `status` should be treated as `High` due detection heuristics despite no shell-outs.

### What should be done in the future
- Add a lightweight complexity report script committed under ticket `scripts/` for repeatable audits.

### Code review instructions
- Review extracted metrics and line-anchored references in the design doc.
- Confirm that each complexity rating has concrete rationale.

### Technical details
- Key helper duplication discovered:
  - `detectWorkspace` and `loadWorkspace` in `cmd_status.go`.
  - `detectCurrentWorkspace` in `cmd_commit.go`.

## Step 3: Design Synthesis and Migration Blueprint

I wrote the primary design document in verbose, intern-oriented form with architecture narrative, complexity matrix, command-layer functionality inventory, extraction strategy, phased implementation plan, API sketches, pseudocode, and diagrams.

The document intentionally separates immediate product-level change (remove `sync`) from structural refactor (service extraction) to keep execution risk manageable.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Produce a detailed architecture guide that can be implemented incrementally without losing the long-term goal.

**Inferred user intent:** Make this refactor executable by an intern without tribal knowledge.

### What I did
- Rewrote `tasks.md` with phase-based execution plan.
- Replaced the design-doc template with a full analysis document containing:
  - Executive summary and problem statement.
  - Full verb complexity table.
  - Current-state command vs pkg analysis.
  - Proposed target architecture and service boundaries.
  - Pseudocode and API reference sketches.
  - Mermaid diagrams (current vs target).
  - Phased migration plan and risk analysis.

### Why
- The requested audience (new intern) needs narrative + concrete implementation hooks.
- Architecture refactors need explicit incremental sequencing.

### What worked
- The resulting plan ties each recommendation to concrete file evidence.
- The migration phases isolate risky changes into manageable cuts.

### What didn't work
- N/A.

### What I learned
- The most important naming fix is decoupling the term `sync` from branch/history services to reduce conceptual debt.

### What was tricky to build
- Balancing "remove sync now" with "preserve branch/log behavior" required explicit split strategy (`BranchOperations`, `HistoryOperations`).

### What warrants a second pair of eyes
- Whether to add a temporary compatibility shim for `wsm sync` or enforce immediate hard removal.

### What should be done in the future
- Convert the proposed API sketches into concrete package files in a follow-up implementation ticket.

### Code review instructions
- Start at the design doc sections:
  - `Verb Complexity Classification`
  - `Current-State Architecture`
  - `Proposed Solution`
  - `Implementation Plan (Phased)`
- Validate references by opening linked files at noted lines.

### Technical details
- Primary output file:
  - `design-doc/01-command-verb-complexity-audit-and-pkg-extraction-plan.md`.

## Step 4: Sprint Replan to Remove 6 Commands and Consolidate

The scope changed from "remove sync" to "remove six commands and consolidate." I converted the ticket checklist into an execution sprint with explicit phases, migration mapping, and commit boundaries. I also updated the design doc with a hard-removal addendum (no backward compatibility) so implementation decisions are unambiguous for handoff.

### Prompt Context

**User prompt (verbatim):** "remove all 6 (including pr + push). And consolidate. Create detailed tasks, update the implementation plan, work off the tasks one after another, commit at appropriate intervals, keep a detailed frequent diary, upload diary to remarkable when done."

**Assistant interpretation:** Switch from analysis-only to implementation mode, execute the six-command removal in phased commits, and maintain a detailed running diary for intern onboarding.

**Inferred user intent:** Reduce command surface aggressively now to simplify architecture and ongoing refactor velocity.

### What I did
- Rewrote `tasks.md` into a phase-based implementation checklist for six-command removal.
- Updated design doc frontmatter/topic hygiene and added sprint addendum for hard command removals.
- Added explicit consolidation mapping from removed verbs to remaining workflows.

### Why
- A strict checklist is needed because execution now spans code deletion, docs migration, build validation, diary publication, and multiple commits.

### What worked
- Existing WSM-MO-005 ticket structure was reusable, so replanning happened in-place without creating extra tickets.

### What didn't work
- N/A.

### What I learned
- The fastest way to avoid scope ambiguity is to codify removed-command replacements directly in tasks and plan docs before touching code.

### What was tricky to build
- Ensuring migration messaging remains practical after dropping both `pr` and `push` from WSM.

### What warrants a second pair of eyes
- Whether future intern work should reintroduce GitHub workflows in `pkg/wsm/workflows` or keep them permanently out-of-scope.

### What should be done in the future
- Add a concise "command deprecation/removal policy" section in repo docs so future command removals follow the same pattern.

### Code review instructions
- Review `tasks.md` Phase 2-5 for sequencing and closure criteria.
- Review sprint update section in the design doc for consolidation mapping.

### Technical details
- Updated docs:
  - `ttmp/.../tasks.md`
  - `ttmp/.../design-doc/01-command-verb-complexity-audit-and-pkg-extraction-plan.md`
