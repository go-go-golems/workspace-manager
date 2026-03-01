---
Title: Investigation diary
Ticket: WSM-MO-010-CLI-ONLY-GIT-BACKEND
Status: complete
Topics:
    - architecture
    - git
    - refactor
    - workspace-manager
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological execution diary for removing gogit/hybrid backends and standardizing WSM on CLI git backend."
LastUpdated: 2026-03-01T10:52:24-05:00
WhatFor: "Track exact migration steps, failures, validations, and commits for the CLI-only backend refactor."
WhenToUse: "Use when reviewing implementation history or continuing work on this backend simplification ticket."
---

# Investigation diary

## Goal

Provide a detailed, step-by-step execution log for the CLI-only backend migration, including concrete commands, failures, decisions, and validation evidence.

## Step 1: Ticket bootstrap, scope confirmation, and execution plan

I created a fresh ticket dedicated to removing `gogit` and `hybrid` backend support with no compatibility layer. I then added a design document and converted the migration scope into granular executable tasks.

This step establishes the sequence for implementation and validation so code changes can be delivered with focused phase commits and traceable evidence.

### Prompt Context

**User prompt (verbatim):** "Create a docmgr tiket for it and add tasks and then work off the tasks and ommit appropriately and keep a detailed diary."

**Assistant interpretation:** Create a new ticket workspace, define granular tasks, then execute the backend migration against those tasks with commit discipline and detailed diary updates.

**Inferred user intent:** Ensure the CLI-only backend refactor is planned and executed transparently with strict documentation hygiene.

**Commit (code):** N/A (planning/setup step)

### What I did

- Created ticket:
  - `docmgr ticket create-ticket --ticket WSM-MO-010-CLI-ONLY-GIT-BACKEND ...`
- Added docs:
  - `design-doc/01-cli-only-git-backend-migration-plan.md`
  - `reference/01-investigation-diary.md`
- Authored task list with implementation phases:
  - backend simplification
  - test/helper cleanup
  - docs/dependency cleanup
  - validation and commit phases
- Authored design document describing:
  - problem statement
  - CLI-only solution
  - rejected alternatives
  - phased implementation plan

### Why

- The user explicitly requested ticket-first execution with task-driven delivery and a detailed diary.

### What worked

- `docmgr` commands succeeded and generated the expected ticket structure and documents.
- Task decomposition maps directly onto concrete code units and validation gates.

### What didn't work

- N/A in this step.

### What I learned

- The existing backend references are spread across runtime wiring, gitclient implementations, tests, integration scenario setup, and architecture docs, so phased execution is necessary to keep changes reviewable.

### What was tricky to build

- The key challenge was defining task granularity that is detailed enough for phased commits without fragmenting the migration into noisy micro-commits.
- Approach: grouped work by architectural boundaries (runtime wiring, implementation deletion, test adaptation, docs/deps, validation).

### What warrants a second pair of eyes

- Task-to-commit mapping once implementation starts, to ensure each phase remains coherent and independently verifiable.

### What should be done in the future

- Execute tasks in order and update this diary after each committed phase.

### Code review instructions

- Start with ticket artifacts:
  - `tasks.md`
  - `design-doc/01-cli-only-git-backend-migration-plan.md`
- Confirm task scope aligns with backend wiring in:
  - `pkg/wsm/git_integration.go`
  - `pkg/wsm/gitclient/*`

### Technical details

- New ticket path:
  - `ttmp/2026/03/01/WSM-MO-010-CLI-ONLY-GIT-BACKEND--remove-gogit-and-hybrid-backends-standardize-on-cli-git-backend`

## Step 2: Execute CLI-only backend migration and validate end-to-end

I completed the core migration by removing `gogit` and `hybrid` client implementations, deleting runtime backend selection, and standardizing backend construction on the CLI implementation only. I also removed integration scenario backend overrides and backend-matrix test loops.

During validation, one integration scenario failed due non-fast-forward push behavior when two repositories intentionally shared the same remote and branch. I fixed the scenario fixture to use separate remotes, then reran full lint/tests successfully.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Execute all migration tasks from the ticket and deliver validated, committed code with detailed progress logging.

**Inferred user intent:** Complete the backend simplification fully, not partially.

**Commit (code):** `bdbf2e6` — "wsm(git): remove gogit/hybrid and standardize on CLI backend"

### What I did

- Runtime/backend wiring:
  - Replaced backend selection in `pkg/wsm/git_integration.go` with CLI-only construction.
  - Removed `WSM_GIT_BACKEND` compatibility behavior from integration helper usage.
- Deleted backend implementations/tests:
  - `pkg/wsm/gitclient/gogit_client.go`
  - `pkg/wsm/gitclient/hybrid_client.go`
  - `pkg/wsm/gitclient/hybrid_client_test.go`
- Updated backend-related tests:
  - Rewrote backend-matrix tests in `pkg/wsm/branch_operations_test.go` and `pkg/wsm/workspace_branch_test.go` to CLI-only.
  - Rewrote gogit-specific assertions in `pkg/wsm/gitclient/remote_branch_exists_test.go` to CLI equivalents.
- Updated integration scenarios:
  - Removed all `s.SetBackend("hybrid")` calls.
  - Removed helper `SetBackend` method in `test/integration/helpers/sandbox.go`.
- Updated docs:
  - Revised `pkg/docs/04-architecture-overview.md` to describe CLI-only gitclient layer.
- Dependency cleanup:
  - Removed `go-git` module references from `go.mod` and stale go-git related entries from `go.sum`.
- Validation:
  - `go test ./... -count=1`
  - `golangci-lint run -v`

### Why

- The ticket goal explicitly required a breaking simplification to CLI-only git backend with no compatibility mode.

### What worked

- Core backend migration compiled immediately after reference cleanup.
- Lint passed after updating all backend-specific test and scenario references.
- Full test suite passed after adjusting one scenario fixture.

### What didn't work

- First full test run failed at `TestCommitPush` with non-fast-forward push rejection:
  - `feature/commit -> feature/commit (fetch first)`
- Root cause: fixture used one remote for two repos, producing divergent commits on the same branch.
- Resolution: switched fixture to separate remotes (`remote1`, `remote2`) in `test/integration/scenarios/commit_push_test.go`.

### What I learned

- CLI-only backend behavior is more explicit about push branch divergence scenarios than prior mixed backend behavior, exposing brittle fixture assumptions.

### What was tricky to build

- The tricky part was avoiding accidental module graph churn while removing `go-git`.
- Symptom: prior runs had produced large, unrelated `go.mod`/`go.sum` rewrites.
- Approach: explicitly dropped go-git requirements and stale go.sum entries without running a broad `go mod tidy` that would rewrite unrelated dependency versions.

### What warrants a second pair of eyes

- Whether any external automation depends on `WSM_GIT_BACKEND` environment behavior that is now removed.

### What should be done in the future

- Optional follow-up: remove or simplify any residual interface comments that imply multi-backend support if that is no longer planned.

### Code review instructions

- Start with:
  - `pkg/wsm/git_integration.go`
  - `pkg/wsm/gitclient/remote_branch_exists_test.go`
  - `test/integration/helpers/sandbox.go`
  - `test/integration/scenarios/commit_push_test.go`
- Confirm deleted files:
  - `pkg/wsm/gitclient/gogit_client.go`
  - `pkg/wsm/gitclient/hybrid_client.go`
  - `pkg/wsm/gitclient/hybrid_client_test.go`
- Re-run:
  - `go test ./... -count=1`
  - `golangci-lint run -v`

### Technical details

- Final code commit for this phase:
  - `bdbf2e6`
