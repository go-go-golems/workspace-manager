---
Title: WSM JS API Completion and Consistency Design
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
    - Path: cmd/wsm/cmds/git/root.go
      Note: |-
        Git command inventory to map into JS surface
        Git command inventory to expose in JS
    - Path: cmd/wsm/cmds/registry/root.go
      Note: |-
        Registry command inventory to map into JS surface
        Registry command inventory to expose in JS
    - Path: cmd/wsm/cmds/workspace/root.go
      Note: |-
        Workspace command inventory to map into JS surface
        Workspace command inventory to expose in JS
    - Path: pkg/docs/03-js-api-and-runner.md
      Note: Existing user-facing JS API documentation
    - Path: pkg/wsm/workflows/commit_workflow.go
      Note: |-
        Commit workflow contract and lifecycle
        Commit API behavior mapping
    - Path: pkg/wsm/workflows/create_workflow.go
      Note: Workspace creation contract
    - Path: pkg/wsm/workflows/delete_workflow.go
      Note: Workspace delete contract
    - Path: pkg/wsm/workflows/fork_workflow.go
      Note: Workspace fork contract
    - Path: pkg/wsm/workflows/info_workflow.go
      Note: Workspace info contract
    - Path: pkg/wsm/workflows/list_workflow.go
      Note: Registry list repos/workspaces contract
    - Path: pkg/wsm/workflows/merge_workflow.go
      Note: Workspace merge contract
    - Path: pkg/wsm/workflows/rebase_workflow.go
      Note: |-
        Rebase/run/status/continue/abort contract
        Rebase API behavior mapping
    - Path: pkg/wsmjs/module/module.go
      Note: |-
        Current require("wsm") export surface and wrapper behavior
        Current and target module API export tree
    - Path: pkg/wsmjs/service/manager.go
      Note: |-
        Current service facade and missing operations
        Service facade expansion point
    - Path: test/integration/scenarios/js_runner_api_scenarios_test.go
      Note: |-
        Current JS integration scenario harness
        Current JS runner scenario baseline
    - Path: test/js/README.md
      Note: |-
        Current JS demo script baseline
        Current and future script matrix
ExternalSources: []
Summary: Completion-level API design for filling all JS surface gaps and enforcing consistent naming, contracts, error semantics, typings, and test coverage.
LastUpdated: 2026-03-01T06:45:00-05:00
WhatFor: Lock down a complete and consistent JS API spec before implementing missing operations.
WhenToUse: Use when implementing or reviewing any new require("wsm") methods, wrappers, typings, or JS demos/tests.
---


# WSM JS API Completion and Consistency Design

## Executive Summary

The initial JS API slice delivered the runner path and a usable baseline (`discover`, `createWorkspace`, `status`, list operations). That solved initial usability, but it is intentionally incomplete relative to workspace-manager capabilities.

This document is the completion design: it defines the full target JS API, consistency rules, error and data contracts, typing strategy, and test/demo plan required to close the remaining gaps without creating a fragmented interface.

Primary outcomes of this design:

1. A complete API inventory covering registry, workspace lifecycle, and git workflows.
2. One consistent namespace and naming model with clear top-level vs manager vs handle responsibilities.
3. Explicit error semantics so scripts are predictable.
4. A concrete TypeScript declaration contract for discoverability and static validation.
5. A phased implementation plan and detailed task backlog including tests and demo scripts.

This document does not replace the initial brainstorm. It narrows and operationalizes it.

## Problem Statement

Current JS module implementation in `pkg/wsmjs/module/module.go` exports only a subset of WSM features.

Implemented today:

1. Top-level: `createManager`, `discover`, `createWorkspace`, `status`, `consts`, `version`.
2. Manager flat methods: `discover`, `createWorkspace`, `status`, `listWorkspaces`, `listRepositories`.
3. Manager namespaces: `registry.listRepositories`, `workspaces.create/list/status`, `git.status`.

Missing relative to current workflow/CLI capabilities:

1. Workspace APIs: `info`, `add`, `remove`, `delete`, `fork`, `merge`.
2. Git APIs: `commit`, `diff`, `log`, `branch.create`, `branch.switch`, `branch.list`, `rebase`, `rebase.status`, `rebase.continue`, `rebase.abort`.
3. Rich workspace handles (`loadWorkspace`, object methods) are absent.
4. Stable `.d.ts` contract is absent.
5. No completion-level demo and integration matrix for these future methods yet.

Without a completion design, implementation may drift in naming, behavior, and result shape across domains.

## Scope

## In Scope

1. Define the completed JS API contract for all currently available workflow domains.
2. Define naming and data consistency rules.
3. Define module/service package architecture changes required.
4. Define typing and test/demo strategy.
5. Define implementation phases and task backlog.

## Out of Scope

1. Building async event streaming API in this phase.
2. Converting runner into a general Node-compatible runtime.
3. Deprecating current baseline methods immediately.

## Design Goals

1. Complete: cover all supported workflow domains already present in Go.
2. Consistent: one naming system across top-level, manager, and namespace methods.
3. Predictable: identical input/output semantics independent of entrypoint alias.
4. Script-first: easy for short scripts and robust for larger automation.
5. Testable: each method family has demo and integration coverage.
6. Evolvable: API can grow without breaking existing scripts.

## API Consistency Rules (Normative)

The following rules are mandatory for new JS methods.

1. JS method names use camelCase (`createWorkspace`, `rebaseStatus` only if not in nested object; prefer nested object methods where possible).
2. Inputs are always objects; no positional optional arguments.
3. Required fields are validated in module layer with explicit type errors.
4. Results are plain objects/arrays with stable field names; avoid command-style text in result payloads.
5. Method aliases (flat + namespaced) must call shared closures to avoid divergence.
6. Operations throw on command-level failures; batch methods include per-repo row errors in data shape.
7. Namespace layout is stable: `registry`, `workspaces`, `git`.
8. Defaulting behavior (`jobs`, branch generation) is centralized in service layer.
9. New methods must be added to `.d.ts` and demo/tests in same PR.

## Proposed Complete Public API

## Top-level Module

```javascript
const wsm = require("wsm");

wsm.version;
wsm.consts;
wsm.createManager(opts);

// convenience wrappers (manager-less)
wsm.discover(input);
wsm.createWorkspace(input);
wsm.status(input);
```

Top-level wrappers remain intentionally small. They are convenience aliases, not the entire API. For full workflows, use manager namespaces.

## Manager Root Methods

```javascript
const manager = wsm.createManager({ defaultJobs: 8 });

manager.discover(input);
manager.createWorkspace(input);
manager.status(input);
manager.listWorkspaces();
manager.listRepositories(input);
manager.loadWorkspace(name);
```

`loadWorkspace(name)` returns a `WorkspaceHandle` (described below).

## Manager Namespaces

```javascript
manager.registry.listRepositories({ tags: [] });
manager.registry.listWorkspaces();

manager.workspaces.create(input);
manager.workspaces.info({ workspaceName });
manager.workspaces.status({ workspaceName, jobs });
manager.workspaces.add({ workspaceName, repoName, branch, force });
manager.workspaces.remove({ workspaceName, repoName, force, removeFiles });
manager.workspaces.delete({ workspaceName, removeFiles, forceWorktrees });
manager.workspaces.fork({ workspaceName, newWorkspaceName, branch, branchPrefix, baseBranch, agentSource, dryRun });
manager.workspaces.merge({ workspaceName, dryRun, keepWorkspace });
manager.workspaces.list();

manager.git.commit({ message, template, addAll, push, dryRun, selectedChanges });
manager.git.diff({ staged, repo, jobs });
manager.git.log({ limit, since, oneline, repo, jobs });
manager.git.branch.create({ branch, track, baseRef, repo, jobs });
manager.git.branch.switch({ branch, repo, jobs });
manager.git.branch.list({ verbose, repo, jobs });
manager.git.rebase.run({ repository, targetBranch, interactive, dryRun, jobs });
manager.git.rebase.status({ repository, jobs });
manager.git.rebase.continue({ repository, jobs });
manager.git.rebase.abort({ repository, jobs });
```

## Workspace Handle API

```javascript
const ws = manager.loadWorkspace("my-workspace");

ws.name();
ws.path();
ws.info();
ws.status({ jobs });
ws.addRepository({ repoName, branch, force });
ws.removeRepository({ repoName, force, removeFiles });
ws.delete({ removeFiles, forceWorktrees });
ws.merge({ dryRun, keepWorkspace });

ws.git.commit({ message, template, addAll, push, dryRun, selectedChanges });
ws.git.diff({ staged, repo, jobs });
ws.git.log({ limit, since, oneline, repo, jobs });
ws.git.branch.create({ branch, track, baseRef, repo, jobs });
ws.git.branch.switch({ branch, repo, jobs });
ws.git.branch.list({ verbose, repo, jobs });
ws.git.rebase.run({ repository, targetBranch, interactive, dryRun, jobs });
ws.git.rebase.status({ repository, jobs });
ws.git.rebase.continue({ repository, jobs });
ws.git.rebase.abort({ repository, jobs });
```

Handle methods are scoped aliases with implicit `workspaceName`.

## Data Contract Design

## Naming Rules

1. JS input/output field names are camelCase.
2. Existing Go-exported fields from marshaled structs may be PascalCase today; completion phase should normalize at JS boundary.
3. If normalization is deferred, add compatibility layer in `.d.ts` and mark future migration.

Recommended target (completion): normalized camelCase DTOs in JS boundary layer.

## Row/Result Shapes

For operations returning multi-repo status/action rows:

1. Use explicit arrays of row objects.
2. Each row includes `repository`, `success` or `state`, and optional `error`.
3. Keep raw counts and booleans (`conflicts`, `ahead`, `behind`) as primitives.

Example:

```javascript
{
  rows: [
    { repository: "repo1", success: true, rebased: true, conflicts: false, error: "" }
  ],
  summary: { total: 1, succeeded: 1, failed: 0 }
}
```

## Error Model

1. Validation errors throw JS `TypeError` in module wrappers.
2. Execution errors throw JS error for call-level failure.
3. Batch partial failures are represented in row payload plus optional summary-level warning fields.
4. Error strings should be stable enough for script assertions (avoid embedding volatile terminal formatting).
5. Future enhancement: add `code` field (`WSM_VALIDATION`, `WSM_NOT_FOUND`, `WSM_GIT_FAILURE`) while keeping message.

## TypeScript Contract Strategy

Completion requires a generated declaration file (or checked-in spec) that includes:

1. Complete manager interfaces.
2. Namespace interfaces.
3. Workspace handle interfaces.
4. Input/result DTO types.
5. Const enum-like literal types.

Initial implementation may use template-based approach mirroring geppetto.

Required outputs:

1. `pkg/wsmjs/spec/wsm.d.ts.tmpl`
2. generation command and checked-in generated artifact (or embedded string used in docs/tests)
3. tests that ensure declaration includes exported method names.

## Internal Architecture Changes

## Service Layer Expansion (`pkg/wsmjs/service`)

Current manager facade only handles discover/create/status/list.

Add methods grouped by domain:

1. Workspace lifecycle:
   - `Info`, `AddRepository`, `RemoveRepository`, `DeleteWorkspace`, `ForkWorkspace`, `MergeWorkspace`, `LoadWorkspace`.
2. Git domain:
   - `Commit`, `Diff`, `Log`, `BranchCreate`, `BranchSwitch`, `BranchList`, `RebaseRun`, `RebaseStatus`, `RebaseContinue`, `RebaseAbort`.
3. Shared helpers:
   - resolve workspace from explicit name or handle context,
   - normalize jobs defaulting,
   - build summary rows for batch operations.

## Module Layer Expansion (`pkg/wsmjs/module`)

1. Add wrappers for all new service methods.
2. Add namespace trees for nested operations (`manager.git.branch.*`, `manager.git.rebase.*`).
3. Implement `loadWorkspace(name)` returning handle object with hidden reference.
4. Reuse wrapper closures between flat and namespaced methods to guarantee parity.
5. Keep required-field validation in one helper per input type.

## Runner Layer (`pkg/wsmjs/runner`)

No API surface changes required for basic completion.

Optional quality additions:

1. Ability to pass script arguments/environment map for richer demos.
2. Optional trace mode to print JS stack traces with Go context.

## Compatibility and Migration

Current scripts rely on existing methods. Completion is additive.

Rules:

1. Keep existing method names and semantics intact.
2. Add new methods without renaming old fields in same release.
3. If DTO field normalization changes are introduced, support dual field names during migration window.
4. Maintain existing `test/js/00-07` scripts; add new scripts rather than rewriting baseline behavior tests.

## Testing and Demo Strategy

## Test Layers

1. Service unit tests (`pkg/wsmjs/service/*_test.go`): business logic and workflow mapping.
2. Module runtime tests (`pkg/wsmjs/module/*_test.go`): export shape, validation behavior, namespace parity.
3. Runner tests (`pkg/wsmjs/runner/*_test.go`): script execution plumbing.
4. Integration scenarios (`test/integration/scenarios/*`): end-to-end sandbox behavior.
5. Demo scripts (`test/js/*`): executable documentation and scenario inputs.

## Demo Script Expansion Plan

Current scripts 00-07 cover baseline API. Add completion scripts in sequential bands.

Band A: workspace lifecycle

1. `08-workspace-info.js`
2. `09-workspace-add-remove.js`
3. `10-workspace-delete.js`
4. `11-workspace-fork-merge.js`

Band B: git operations

1. `12-git-commit.js`
2. `13-git-diff.js`
3. `14-git-log.js`
4. `15-git-branch-create-switch-list.js`

Band C: rebase family

1. `16-git-rebase-run-happy.js`
2. `17-git-rebase-status.js`
3. `18-git-rebase-continue.js`
4. `19-git-rebase-abort.js`

Band D: handle and parity

1. `20-workspace-handle-basics.js`
2. `21-workspace-handle-git.js`
3. `22-flat-vs-namespace-parity-extended.js`

## Integration Scenario Expansion Plan

Add focused scenario files rather than one giant test.

1. `js_runner_workspace_lifecycle_scenarios_test.go`
2. `js_runner_git_ops_scenarios_test.go`
3. `js_runner_rebase_scenarios_test.go`
4. `js_runner_workspace_handle_scenarios_test.go`

Each scenario should:

1. Create deterministic sandbox repos/remotes.
2. Execute specific script set via `wsm runner --output-mode data --output json --print-result=false`.
3. Assert row-level and script-level contracts.
4. Avoid dependence on external environment state.

## Design Decisions

## Decision 1: Keep convenience wrappers small

Convenience wrappers remain only for common flow (`discover`, `createWorkspace`, `status`). Full API surface lives on manager namespaces.

Rationale:

1. Keeps top-level uncluttered.
2. Encourages explicit structure for advanced operations.

## Decision 2: Add workspace handles now

Rationale:

1. Needed for consistent ergonomics in multi-step scripts.
2. Aligns with initial design and geppetto-like patterns.

## Decision 3: Nested git namespaces (`git.branch`, `git.rebase`)

Rationale:

1. Prevents method-name collisions.
2. Improves discoverability and conceptual grouping.

## Decision 4: Throw-first error model with batch rows

Rationale:

1. Fits JS expectations.
2. Avoids envelope noise for common calls.
3. Still supports partial failures for batch operations.

## Decision 5: Require `.d.ts` parity in same change set

Rationale:

1. Prevents undocumented API drift.
2. Improves IDE discoverability and contributor velocity.

## Alternatives Considered

## Alternative A: Command-wrapper API (`wsm.run("cmd", opts)`)

Rejected:

1. Leaks command UX concerns into API.
2. Harder typing and discoverability.

## Alternative B: Top-level-only giant API

Rejected:

1. Flat namespace becomes noisy and unstable.
2. Poor onboarding and method grouping.

## Alternative C: Handle-only API (no manager namespaces)

Rejected:

1. Overkill for scripts that do not need handles.
2. Removes useful functional/namespace-style access.

## Alternative D: Postpone typings until end

Rejected:

1. High drift risk.
2. Harder to review API consistency.

## Implementation Plan (Phased)

## Phase 5A: Contract Lock and Scaffolding

1. Finalize completion API signatures and DTO schemas.
2. Add stub interfaces/types in service package.
3. Add module export skeletons (failing tests acceptable in WIP branch only).

Exit criteria:

1. API spec accepted and documented.
2. New module export names reserved.

## Phase 5B: Workspace Lifecycle APIs

1. Implement service methods for info/add/remove/delete/fork/merge/loadWorkspace.
2. Implement module wrappers under `manager.workspaces.*` and handle methods.
3. Add module tests for validation and method presence.
4. Add scripts 08-11 and lifecycle scenario tests.

Exit criteria:

1. Lifecycle methods runnable from JS.
2. Scripts/scenarios pass.

## Phase 5C: Git Core APIs (commit/diff/log/branch)

1. Implement service methods for commit/diff/log/branch operations.
2. Expose methods under `manager.git.*` and `workspaceHandle.git.*`.
3. Add scripts 12-15.
4. Add scenario tests for commit/diff/log/branch.

Exit criteria:

1. Git core methods stable and tested.

## Phase 5D: Rebase Family APIs

1. Implement service wrappers for rebase run/status/continue/abort.
2. Expose under `manager.git.rebase.*` and handle aliases.
3. Add scripts 16-19.
4. Add dedicated rebase JS scenarios matching existing CLI conflict/happy patterns.

Exit criteria:

1. Rebase JS flows pass both happy and conflict scenarios.

## Phase 5E: Typings, Docs, and Parity Hardening

1. Implement `.d.ts` generation/spec and include all new exports.
2. Add scripts 20-22 for handle usage and parity checks.
3. Update docs (`pkg/docs/03-js-api-and-runner.md`) with full API matrix.
4. Run comprehensive test suite.

Exit criteria:

1. JS docs, typings, demos, and tests all align with code.

## Open Questions

1. Should commit API include interactive selection support in JS or require explicit `selectedChanges` only?
2. Should handle methods mutate cached state or always resolve latest state from workflows?
3. Should we expose explicit summary objects for all batch methods from day one, or add later?
4. Do we want a lightweight deprecation policy/version field beyond `version: "0.1.0"` once surface expands significantly?

## References

1. `pkg/wsmjs/module/module.go`
2. `pkg/wsmjs/service/manager.go`
3. `pkg/wsm/workflows/create_workflow.go`
4. `pkg/wsm/workflows/delete_workflow.go`
5. `pkg/wsm/workflows/info_workflow.go`
6. `pkg/wsm/workflows/fork_workflow.go`
7. `pkg/wsm/workflows/merge_workflow.go`
8. `pkg/wsm/workflows/commit_workflow.go`
9. `pkg/wsm/workflows/rebase_workflow.go`
10. `pkg/wsm/workflows/list_workflow.go`
11. `cmd/wsm/cmds/workspace/root.go`
12. `cmd/wsm/cmds/git/root.go`
13. `cmd/wsm/cmds/registry/root.go`
14. `pkg/docs/03-js-api-and-runner.md`
15. `test/js/README.md`
16. `test/integration/scenarios/js_runner_api_scenarios_test.go`
17. `ttmp/2026/02/28/WSM-MO-007-JS-API-DESIGN--design-javascript-api-for-workspace-manager/design-doc/01-workspace-manager-javascript-api-brainstorm-and-design-options.md`
18. `ttmp/2026/02/28/WSM-MO-007-JS-API-DESIGN--design-javascript-api-for-workspace-manager/design-doc/02-js-scripting-demo-suite-implementation-plan-and-scenario-mapping.md`
