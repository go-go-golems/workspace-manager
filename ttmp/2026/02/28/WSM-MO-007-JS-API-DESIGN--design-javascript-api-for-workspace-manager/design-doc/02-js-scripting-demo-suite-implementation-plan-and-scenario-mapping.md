---
Title: JS Scripting Demo Suite Implementation Plan and Scenario Mapping
Ticket: WSM-MO-007-JS-API-DESIGN
Status: active
Topics:
    - architecture
    - api-design
    - workspace-manager
    - javascript
    - goja
    - geppetto
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/wsm/cmds/js/runner.go
      Note: |-
        CLI execution path for script runner tests
        CLI runner command behavior
    - Path: pkg/wsmjs/module/module.go
      Note: |-
        Native require("wsm") API implementation
        Native module exports used by scripts
    - Path: pkg/wsmjs/service/manager.go
      Note: |-
        Go facade mapping scripts to workflows
        Service facade path from JS calls
    - Path: test/integration/scenarios/js_runner_api_scenarios_test.go
      Note: |-
        Integration harness that executes demo scripts via wsm runner
        Integration runner execution harness
    - Path: test/js/00-module-surface.js
      Note: |-
        Baseline module/export smoke demo
        Module surface contract demo
    - Path: test/js/01-discover-and-list.js
      Note: |-
        Discover workflow plus repository listing demo
        Discovery plus repository listing demo
    - Path: test/js/02-create-workspace.js
      Note: |-
        Workspace creation demo with explicit branch behavior
        Workspace creation behavior demo
    - Path: test/js/03-status-namespace-parity.js
      Note: |-
        Namespace parity checks for status APIs
        Namespace parity checks for status
    - Path: test/js/04-convenience-lifecycle.js
      Note: |-
        Top-level convenience API lifecycle demo
        Top-level convenience lifecycle demo
    - Path: test/js/05-list-repository-parity.js
      Note: |-
        Flat versus grouped list API parity demo
        Flat versus grouped list parity
    - Path: test/js/06-validation-errors.js
      Note: |-
        JS-facing validation and exception handling demo
        JS validation and exception behavior demo
    - Path: test/js/07-default-jobs-status.js
      Note: |-
        Manager defaultJobs behavior demo
        defaultJobs manager option demo
    - Path: test/js/README.md
      Note: Script catalog and manual execution notes
ExternalSources: []
Summary: Detailed, intern-oriented implementation plan for building and validating a comprehensive JS demo suite under test/js and running it through integration scenarios.
LastUpdated: 2026-03-01T06:06:00-05:00
WhatFor: Provide a complete onboarding-quality blueprint for understanding, executing, and extending JS scripting demos tied to integration scenarios.
WhenToUse: Use when onboarding contributors to the WSM JS API, maintaining test/js demos, or debugging runner/integration regressions.
---


# JS Scripting Demo Suite Implementation Plan and Scenario Mapping

## Executive Summary

This document explains, in implementation-level detail, how to build and maintain a complete JavaScript demo suite under `test/js/`, how those scripts validate the current `require("wsm")` API, and how integration scenarios execute those scripts as part of normal test runs.

The goal is twofold:

1. Make the JS API concrete and discoverable through runnable examples instead of only prose docs.
2. Ensure demos are not "toy-only" artifacts; they must be verified continuously inside integration tests that use realistic repository/workspace setups.

The final structure includes eight demo scripts that cover module surface, discovery, workspace creation, status parity across namespaces, convenience wrappers, repository listing parity, validation error behavior, and manager default configuration behavior. Integration tests run these scripts in sandboxed environments using the same helper stack used by other scenario tests.

This plan is intentionally intern-oriented. It explains not only what each file does, but why it exists, what assumptions it makes, how to execute it manually, which scenario preconditions it needs, and how to tell whether output indicates success, regression, or partial misconfiguration.

## Problem Statement

The JS API for Workspace Manager already existed in code (`pkg/wsmjs/module` + `pkg/wsmjs/service` + `wsm runner`), but there was a gap between API availability and API confidence:

1. We had a basic smoke script in `demo/js/`, but not a full catalog of examples that covers the API surface area now exposed.
2. We lacked a dedicated `test/js/` script suite designed for test repeatability and scenario-based validation.
3. Existing integration scenarios mostly validated CLI commands directly; they did not systematically validate the JS runner path using the same repository/workspace arrangements.
4. We did not yet have intern-friendly, step-by-step guidance mapping each script to expected integration behavior.

That gap creates risk:

1. API regressions can ship unnoticed if only Go unit tests and CLI tests are run.
2. New contributors cannot quickly understand the difference between manager-level, grouped, and convenience API calls.
3. Script examples can drift from real behavior if they are not continuously executed in CI.

The request for this ticket phase is to close that gap comprehensively.

## Scope and Non-Goals

## In Scope

1. Create a full set of demo scripts in `test/js/` that exercise the current JS API surface implemented today.
2. Build integration tests that set up real sandbox repos/workspaces and execute those scripts through `wsm runner`.
3. Assert script outputs and expected behavior from integration tests.
4. Provide a detailed implementation-and-onboarding document (this file).
5. Tie the work into ticket bookkeeping and reMarkable delivery.

## Out of Scope

1. Expanding the JS API to include non-implemented operations (for example commit/rebase wrappers) in this change.
2. Rewriting existing non-JS integration scenario files unless required for compatibility.
3. Building a TypeScript declaration generator in this phase.
4. Converting runner to accept dynamic script parameters from CLI flags.

## Current Architecture You Need to Understand First

Before touching scripts, an intern should understand the execution path.

## API and Runtime Stack

1. `wsm runner <script.js>` invokes the runner command at `cmd/wsm/cmds/js/runner.go`.
2. That command calls `pkg/wsmjs/runner.RunFile`, which spins up a goja VM and `require` registry.
3. The runner registers the native module from `pkg/wsmjs/module` under `require("wsm")`.
4. JS method calls (for example `manager.createWorkspace`) are forwarded to `pkg/wsmjs/service.Manager`.
5. Service methods call existing workflows in `pkg/wsm/workflows/*` (discover/create/status/list).

This layering matters for debugging:

1. If script load fails, the issue is likely in runner command or file path.
2. If `require("wsm")` fails, the issue is module registration.
3. If API call shape fails, the issue is decode/validation in `module.go`.
4. If API call logic fails after decode, the issue is service/workflow/domain behavior.

## API Surface Supported Today

Current scriptable operations available now:

1. `wsm.createManager({ defaultJobs })`
2. Top-level convenience wrappers:
   - `wsm.discover(input)`
   - `wsm.createWorkspace(input)`
   - `wsm.status(input)`
3. Manager flat methods:
   - `manager.discover`
   - `manager.createWorkspace`
   - `manager.status`
   - `manager.listWorkspaces`
   - `manager.listRepositories`
4. Manager grouped namespaces:
   - `manager.registry.listRepositories`
   - `manager.workspaces.create`
   - `manager.workspaces.list`
   - `manager.workspaces.status`
   - `manager.git.status`
5. Constants in `wsm.consts.*`

Important limitation: no `commit`, `rebase`, `delete`, or conflict APIs are exposed on JS module yet. Scripts and tests in this phase intentionally stay aligned with implemented surface.

## Demo Suite Design Principles

The `test/js/` suite is structured with strict principles:

1. Every script is executable as a standalone demo.
2. Every script ends with an object literal so the runner can emit a structured result.
3. Every script contains in-script assertions (`assert(...)`) and throws on mismatch.
4. Integration tests assert both runner-level success and script-level semantic success (`result.ok === true`).
5. Scripts are deterministic against controlled sandbox setups from integration helpers.

This gives two levels of confidence:

1. The runner path works.
2. The business behavior expected by the script works.

## Script Catalog and Responsibilities

## `test/js/00-module-surface.js`

Purpose:

1. Verify that `require("wsm")` loads.
2. Verify expected top-level exports and grouped manager methods.
3. Verify key constants that should remain stable (`origin`, resolution mode values).

Why it matters:

1. Catches breaking API shape changes immediately.
2. Provides the fastest smoke check for a contributor new to JS API.

Expected success signals:

1. `result.script === "00-module-surface"`
2. `result.ok === true`
3. `managerSurface` fields are all `"function"`.

Typical failure modes:

1. Missing `createManager` export after refactor.
2. Constant renaming that breaks script consumers.
3. Namespace objects not installed.

## `test/js/01-discover-and-list.js`

Purpose:

1. Run discovery against current working directory (`paths: ["."]`).
2. Validate discovered repository count.
3. Validate repository listing through grouped namespace (`manager.registry.listRepositories`).

Why it matters:

1. Shows discover -> list workflow expected in automation scripts.
2. Verifies that discovery side effects are persisted and list API reflects them.

Expected success signals:

1. `RepositoryCount >= 2` in two-repo scenario.
2. List API returns at least the same two repos.

Typical failure modes:

1. Wrong execution working directory (script sees no repos).
2. Registry persistence path mismatch in sandbox environment.
3. Discover defaults changed unexpectedly.

## `test/js/02-create-workspace.js`

Purpose:

1. Create workspace `ws-js-demo` from `repo1` + `repo2`.
2. Use explicit branch (`feature/js-demo`) to test non-auto branch path.
3. Verify workspace appears in `manager.workspaces.list()`.

Why it matters:

1. Validates create workflow through JS API, not only CLI command path.
2. Demonstrates post-create verification pattern interns should follow.

Expected success signals:

1. `created.Workspace.name === "ws-js-demo"`
2. `created.FinalBranch === "feature/js-demo"`
3. `autoBranchGenerated === false`
4. list includes `ws-js-demo`

Typical failure modes:

1. Repo names not discovered before create.
2. Branch plumbing regression in create workflow.
3. Workspace list not updated after creation.

## `test/js/03-status-namespace-parity.js`

Purpose:

1. Query status through three equivalent entrypoints:
   - `manager.status`
   - `manager.workspaces.status`
   - `manager.git.status`
2. Assert equal repository name sets.

Why it matters:

1. Confirms grouped namespaces are aliases of shared logic, not divergent implementations.
2. Prevents accidental skew between flat and grouped API behavior.

Expected success signals:

1. All three calls resolve workspace `ws-js-demo`.
2. All three produce identical repository sets.
3. `repositoryCount === 2` in lifecycle scenario.

Typical failure modes:

1. One namespace wired to wrong function.
2. Input decode mismatch in one wrapper function.
3. Accidental future divergence after adding namespace-specific logic.

## `test/js/04-convenience-lifecycle.js`

Purpose:

1. Exercise top-level convenience API (`wsm.discover`, `wsm.createWorkspace`, `wsm.status`).
2. Validate auto-branch generation path (`branchPrefix: "feat"`, branch omitted).

Why it matters:

1. Demonstrates minimal script style for short automations.
2. Confirms convenience wrappers remain functionally equivalent to manager-based calls.

Expected success signals:

1. `created.AutoBranchGenerated === true`
2. `created.FinalBranch === "feat/ws-js-convenience"`
3. Status repository count equals 1 in one-repo setup.

Typical failure modes:

1. Branch generation semantics changed.
2. Convenience wrappers not using same defaults as manager.
3. Workspace lookup fails if script and scenario names drift.

## `test/js/05-list-repository-parity.js`

Purpose:

1. Compare `manager.listRepositories` versus `manager.registry.listRepositories`.
2. Assert row counts and sorted names are identical.

Why it matters:

1. Documents dual access patterns clearly.
2. Guards against subtle divergence during future refactors.

Expected success signals:

1. Parity boolean true.
2. Count >= 2 in lifecycle scenario.

Typical failure modes:

1. Flat and grouped wrappers passing different parameters.
2. Hidden filtering differences introduced accidentally.

## `test/js/06-validation-errors.js`

Purpose:

1. Demonstrate JS-visible validation errors.
2. Verify stable error messages for missing required fields.

Why it matters:

1. Teaches intern how to write scripts that expect and handle exceptions.
2. Catches accidental message or validation contract regressions.

Expected success signals:

1. Missing name error includes `createWorkspace requires name`.
2. Missing repos error includes `createWorkspace requires repos`.

Typical failure modes:

1. Validation removed unintentionally.
2. Error message changed without test updates.

## `test/js/07-default-jobs-status.js`

Purpose:

1. Demonstrate manager-level `defaultJobs` by omitting jobs in status call.
2. Verify status still resolves expected workspace.

Why it matters:

1. Encodes how global manager options should be used by script authors.
2. Provides guardrail for default option plumbing.

Expected success signals:

1. Workspace resolves as `ws-js-demo`.
2. Repository count remains 2.

Typical failure modes:

1. Manager option decode breaks.
2. Defaults not propagated to service layer.

## Integration Scenario Mapping

This section explains exactly how scripts are wired into integration tests and why each setup step exists.

## Scenario File

Primary integration harness file:

- `test/integration/scenarios/js_runner_api_scenarios_test.go`

It contains two high-level scenarios.

## Scenario 1: `TestJSRunnerDemoScriptsLifecycleScenario`

Setup:

1. Create sandbox.
2. Set backend to `hybrid`.
3. Initialize one bare remote.
4. Initialize `repo1` and `repo2` against that remote.

Execution order and rationale:

1. Run `00-module-surface.js` first.
   - No repository state dependency.
   - Fails fast if module export shape regresses.
2. Run `01-discover-and-list.js` in `ReposDir`.
   - Needs `paths: ["."]` to target sandbox repos.
   - Persists discovered repos into registry.
3. Run `05-list-repository-parity.js`.
   - Requires discovery already completed.
4. Run `02-create-workspace.js`.
   - Depends on discovered repo names (`repo1`, `repo2`).
5. Run `03-status-namespace-parity.js`.
   - Depends on workspace `ws-js-demo` existing.
6. Run `07-default-jobs-status.js`.
   - Also depends on same created workspace.
7. Run `06-validation-errors.js`.
   - Independent, but kept in same test as API behavior validation block.

Assertions made by Go harness:

1. Runner command exit code is zero.
2. Output-mode data JSON row parses correctly.
3. Row-level `status == "ok"` and `has_result == true`.
4. Script-level `result.ok == true`.
5. Script-specific result fields match expected values.

Expected final result:

1. Test passes in ~2-4 seconds locally (excluding initial compile overhead).
2. Each script contributes explicit proof of one API behavior category.

## Scenario 2: `TestJSRunnerDemoScriptsConvenienceScenario`

Setup:

1. Create sandbox.
2. Set backend to `hybrid`.
3. Initialize one bare remote.
4. Initialize `repo1` against that remote.

Execution order and rationale:

1. Run `04-convenience-lifecycle.js` in `ReposDir`.
   - Needs discovery rooted at `.`.
   - Creates workspace `ws-js-convenience` using top-level wrapper.
2. Validate branch auto-generation and repository count.

Expected final result:

1. Demonstrates convenience API lifecycle in a minimal single-repo environment.
2. Ensures convenience wrappers stay valid independently of manager namespace tests.

## Why We Parse Runner Output the Way We Do

Integration harness uses `--output-mode data --output json --print-result=false`. This is intentional:

1. Human output mode mixes status text and JSON pretty-printing and is harder to parse deterministically.
2. Data mode yields structured rows with stable shape.
3. We still defensively parse from the first `[` or last `\n[` to tolerate incidental surrounding logs.

The parser strategy mirrors existing workflow-heavy tests that already account for mixed output behavior in some commands.

## Expected Output Contracts

For intern debugging, below are the key row and result contracts.

## Runner Row Contract

Each successful script execution in data mode should include one row with fields:

1. `script` (path provided to command)
2. `result` (script return object)
3. `has_result` (boolean)
4. `status` (`"ok"`)

If `status != "ok"` or row count differs from 1, treat as harness/runner contract regression.

## Script Result Contract

Each script should return object shape:

1. `ok: true`
2. `script: <script-id>`
3. Scenario-specific fields (counts, branch names, parity booleans, etc.)

If `ok` is false or missing, treat as script semantic regression even if runner command did not fail.

## Expected Values by Script in Current Scenarios

Lifecycle scenario expected values:

1. `00-module-surface`:
   - `consts.origin == "origin"`
2. `01-discover-and-list`:
   - `repositoryCountFromDiscover >= 2`
   - `repositoryCountFromList >= 2`
3. `02-create-workspace`:
   - `workspace == "ws-js-demo"`
   - `finalBranch == "feature/js-demo"`
4. `03-status-namespace-parity`:
   - `parity.rootVsWorkspaces == true`
   - `parity.rootVsGit == true`
5. `05-list-repository-parity`:
   - `parity == true`
6. `06-validation-errors`:
   - message contains `requires name`
   - message contains `requires repos`
7. `07-default-jobs-status`:
   - `repositoryCount == 2`

Convenience scenario expected values:

1. `04-convenience-lifecycle`:
   - `workspace == "ws-js-convenience"`
   - `finalBranch == "feat/ws-js-convenience"`
   - `repositoryCount == 1`

## How to Exercise the Scripts Manually

There are two manual modes: ad-hoc exploratory and scenario-faithful.

## Ad-Hoc Exploratory Mode

Useful when you only want to inspect API behavior quickly.

Example commands:

```bash
wsm runner test/js/00-module-surface.js
wsm runner test/js/06-validation-errors.js --output-mode data --output json
```

Caveat:

1. Scripts that expect specific repositories/workspaces (`repo1`, `repo2`, `ws-js-demo`) may fail unless you prepare matching state.

## Scenario-Faithful Mode (Recommended)

Use integration tests to guarantee setup is correct:

```bash
go test ./test/integration/scenarios -run 'TestJSRunnerDemoScripts' -v
```

This mode is what CI should trust because it sets isolated HOME/config/repo layouts.

## Full Regression Run

When touching runner/module/integration helpers:

```bash
go test ./test/integration/scenarios
```

This verifies both old scenarios and JS script scenarios together.

## Intern Runbook: Debugging Failures Step by Step

When a script scenario fails, do not guess. Use this sequence.

## Step 1: Identify Failure Layer

1. If command exits non-zero before JSON parse, inspect runner command stderr first.
2. If JSON parse fails, inspect stdout for mixed output and confirm command used data mode.
3. If row parses but `result.ok` false, inspect script assertions.
4. If script assertions fail due data mismatch, inspect module/service/workflow behavior.

## Step 2: Re-run Single Scenario Verbosely

```bash
go test ./test/integration/scenarios -run 'TestJSRunnerDemoScriptsLifecycleScenario' -v
```

The test logs each `wsm runner` command and working directory. This is crucial because discovery scripts depend on working directory.

## Step 3: Re-run the Script in Isolation Against Same Sandbox State

Inside test code, copy the failing command from test logs. If needed, temporarily add additional script return fields or `console.log` diagnostics.

## Step 4: Validate API Shape First

If anything odd happens, run `00-module-surface.js` first. It is intentionally the quickest contract check.

## Step 5: Validate Prerequisites

Common prerequisite misses:

1. Discovery was not run before create/list parity checks.
2. Workspace names in script changed without scenario updates.
3. Repos count in setup no longer matches script assumptions.

## Step 6: Decide Whether Failure is Intentional API Change

If behavior changed intentionally:

1. Update script assertion.
2. Update Go scenario assertion.
3. Update this plan document expected values.
4. Update diary/changelog so future readers understand why.

## Design Decisions and Rationale

## Decision 1: Use dedicated `test/js/` folder instead of expanding `demo/js/`

Rationale:

1. `demo/js/` is useful for human examples, but not tightly tied to CI guarantees.
2. `test/js/` signals that scripts are test artifacts with deterministic expectations.
3. Keeps demonstration and regression-oriented examples conceptually separated.

## Decision 2: Keep scripts self-contained (no shared JS helper module)

Rationale:

1. Avoids dependency on local JS module resolution behavior inside goja runner.
2. Makes each script copy-paste runnable.
3. Slight duplication is acceptable for clarity and stability.

## Decision 3: Validate both row-level and script-level success

Rationale:

1. Runner could succeed while script logic silently returns wrong object.
2. Script could throw before returning result while row-level status catches command failure.
3. Two-level assertion model reduces blind spots.

## Decision 4: Use absolute script paths in integration tests

Rationale:

1. Some scripts run with `workDir = s.ReposDir`; relative script paths would break.
2. Absolute paths remove ambiguity and improve reliability across environments.

## Decision 5: Keep scope aligned with currently implemented JS API

Rationale:

1. Prevents fake tests for methods that do not exist.
2. Keeps this phase focused on confidence, not API expansion.
3. Future phase can add scripts when new JS methods ship.

## Alternatives Considered

## Alternative A: Add only one big smoke script

Rejected because:

1. Hard to isolate failures.
2. Poor onboarding value.
3. Too much hidden coupling among checks.

## Alternative B: Add JS tests only in Go unit tests (no integration scenarios)

Rejected because:

1. Unit tests cannot easily model realistic sandbox repo/workspace lifecycle.
2. Would miss runner CLI path and data-mode output behavior.

## Alternative C: Parameterize scripts with CLI args/env immediately

Rejected for this phase because:

1. Current runner does not expose a first-class argument interface to scripts.
2. Adds complexity and new API design questions unrelated to immediate coverage goal.

## Alternative D: Modify all existing scenario tests to call JS equivalents

Rejected because:

1. Current JS API does not yet expose all scenario command surfaces.
2. Would create partial/inconsistent conversion and blur responsibility.
3. Better to add dedicated JS scenarios now and expand incrementally as API grows.

## Detailed Implementation Plan

This is the concrete execution plan used for this phase and reusable for future updates.

## Phase 1: Script authoring in `test/js/`

1. Create script list and assign one behavioral objective to each script.
2. Add a local `assert` helper in each script to fail fast and clearly.
3. Ensure script returns final object literal with `ok` and `script` fields.
4. Keep all scripts ES5-compatible style (`var`, function declarations) to match goja constraints safely.
5. Add `test/js/README.md` to explain script purposes and manual execution.

Definition of done for phase 1:

1. All scripts execute without syntax/runtime errors in intended scenarios.
2. Each script documents a distinct API contract.

## Phase 2: Integration harness wiring

1. Create `js_runner_api_scenarios_test.go` under scenario package.
2. Add helper to locate module root from test working directory.
3. Add helper to run runner command in data mode with absolute script path.
4. Add helper to parse JSON rows robustly.
5. Add helper to coerce numeric fields (`float64` from JSON decode) to ints for assertions.
6. Implement lifecycle scenario and convenience scenario with explicit state setup.
7. Assert script-specific fields after each execution.

Definition of done for phase 2:

1. `go test ./test/integration/scenarios -run 'TestJSRunnerDemoScripts' -v` passes.
2. Failure messages are clear and actionable.

## Phase 3: Validation and regression pass

1. Run targeted JS scenario tests.
2. Run full integration scenario suite.
3. Confirm no regressions introduced.
4. Record command outputs in diary/changelog summary.

Definition of done for phase 3:

1. Full integration scenario package passes with new tests included.

## Phase 4: Documentation and handoff

1. Write this detailed implementation plan doc.
2. Update diary with this implementation step.
3. Update ticket changelog with delivered artifacts and test evidence.
4. Update tasks to include and mark completion for JS demo suite phase.
5. Run `docmgr doctor` to ensure ticket quality gates pass.
6. Upload updated ticket bundle to reMarkable and verify remote listing.

Definition of done for phase 4:

1. Ticket docs + code + upload artifacts are aligned and review-ready.

## How This Supports Future API Expansion

When new JS APIs are added (for example `rebase`, `commit`, `delete`), extend this model:

1. Add one script per new behavior family in `test/js/`.
2. Add scenario setup matching prerequisites for that operation.
3. Add script-level assertions for both happy path and expected error path.
4. Add Go-level assertions on returned structured fields.
5. Update this plan's script matrix and expected output section.

This keeps demos and regressions synchronized over time.

## Risks and Mitigations

## Risk 1: Scripts become stale as API evolves

Mitigation:

1. Keep scripts in CI-backed integration tests.
2. Treat failing script assertions as contract signal, not nuisance.

## Risk 2: Hardcoded names conflict with future scenario changes

Mitigation:

1. Keep names isolated per sandbox test (`ws-js-demo`, `ws-js-convenience`).
2. If parameterization becomes necessary, add runner argument/env support in dedicated follow-up ticket.

## Risk 3: Output parsing breaks if runner row format changes

Mitigation:

1. Update parser helper and row assertions together in one commit.
2. Keep parser tolerant to leading non-JSON lines.

## Risk 4: Contributors misinterpret convenience vs manager APIs

Mitigation:

1. Keep separate scripts for manager namespace parity and convenience lifecycle.
2. Retain explicit comments in scripts and this document.

## Risk 5: Overlapping behavior with existing scenario tests causes redundancy fatigue

Mitigation:

1. JS scenarios focus on API path differences, not re-testing every CLI behavior.
2. Avoid duplicating unrelated heavy workflows until JS API exposes them.

## Review Checklist for New Contributors

Before opening a PR touching JS API, complete this checklist:

1. Ran `go test ./test/integration/scenarios -run 'TestJSRunnerDemoScripts' -v`.
2. Ran `go test ./test/integration/scenarios`.
3. Updated/added `test/js/` scripts if API surface changed.
4. Updated Go scenario assertions for changed result shape.
5. Updated this implementation-plan document expected outputs section.
6. Updated diary/changelog entries under ticket docs.

## Open Questions

1. Should runner grow first-class script argument support (for example `--arg key=value`) to reduce hardcoded test script values?
2. Should we generate a single JS assertion utility injected by runner, or continue with self-contained script assertions?
3. When rebase/commit APIs are added to module, do we keep parity checks in one file or split by domain area (registry/workspaces/git)?

These questions do not block current implementation but matter for next expansion phase.

## References

1. `pkg/wsmjs/module/module.go`
2. `pkg/wsmjs/service/manager.go`
3. `pkg/wsmjs/runner/runner.go`
4. `cmd/wsm/cmds/js/runner.go`
5. `test/js/README.md`
6. `test/js/00-module-surface.js`
7. `test/js/01-discover-and-list.js`
8. `test/js/02-create-workspace.js`
9. `test/js/03-status-namespace-parity.js`
10. `test/js/04-convenience-lifecycle.js`
11. `test/js/05-list-repository-parity.js`
12. `test/js/06-validation-errors.js`
13. `test/js/07-default-jobs-status.js`
14. `test/integration/scenarios/js_runner_api_scenarios_test.go`
15. `test/integration/helpers/sandbox.go`
16. `test/integration/helpers/wsm.go`
17. `ttmp/2026/02/28/WSM-MO-007-JS-API-DESIGN--design-javascript-api-for-workspace-manager/design-doc/01-workspace-manager-javascript-api-brainstorm-and-design-options.md`
18. `ttmp/2026/02/28/WSM-MO-007-JS-API-DESIGN--design-javascript-api-for-workspace-manager/reference/01-investigation-diary.md`
