# Tasks

## Phase 0: Research/Design (Completed)

- [x] Create ticket workspace and initialize design + diary docs
- [x] Map current workspace-manager reusable package and workflow architecture
- [x] Analyze go-go-goja native module/runtime patterns relevant to JS API design
- [x] Analyze geppetto JS API/export/typing patterns for reusable design motifs
- [x] Write design brainstorm doc with multiple API styles, examples, tradeoffs, and implementation hints
- [x] Write detailed chronological diary entries with commands, errors, and decisions
- [x] Run doc validation, relate key files, and upload deliverables to reMarkable

## Phase 1: JS API Core

- [x] Create `pkg/wsmjs/service` facade with request/result contracts over existing workflows
- [x] Implement `pkg/wsmjs/module` with `require("wsm")`, `createManager`, functional shortcuts, and typed const exports
- [x] Add focused module tests proving `require("wsm")` loads and exposes expected API surface

## Phase 2: Runner Verb

- [x] Add reusable script runtime helper under `pkg/wsmjs/runner` to execute JS files with the `wsm` module pre-registered
- [x] Add `wsm runner` CLI verb in `cmd/wsm/cmds/js` (dual-output support + script execution)
- [x] Wire runner command registration into `cmd/wsm/root.go`

## Phase 3: Validation and Documentation

- [x] Run formatting and tests for touched packages
- [x] Update `WSM-MO-007` diary with implementation steps, failures, and review instructions
- [x] Update changelog/task states and upload refreshed deliverable to reMarkable

## Phase 4: JS Demo Suite and Scenario Coverage

- [x] Create a full `test/js/` demo script suite that showcases current `require("wsm")` API surfaces
- [x] Add integration scenario tests that execute `test/js` scripts via `wsm runner` in sandboxed environments
- [x] Validate the new JS scenario tests and full integration scenario package
- [x] Author a detailed intern-facing 7+ page implementation plan documenting script behavior, scenario mapping, and expected results
- [x] Upload refreshed ticket bundle containing the new implementation plan to reMarkable

## Phase 5A: Completion Contract Lock

- [x] Review and approve completion design doc `design-doc/03-wsm-js-api-completion-and-consistency-design.md`
- [x] Freeze final JS method names/signatures for manager, namespaces, and workspace handles
- [x] Define canonical JS DTO naming rules (camelCase target + compatibility strategy for existing fields)
- [x] Define standardized JS error model (`TypeError` validation, throw-first execution, batch row errors)
- [x] Add/update ticket changelog and diary entries documenting any design adjustments from review

## Phase 5B: Service Layer Expansion (`pkg/wsmjs/service`)

- [x] Add workspace lifecycle service methods: `Info`, `AddRepository`, `RemoveRepository`, `DeleteWorkspace`, `ForkWorkspace`, `MergeWorkspace`, `LoadWorkspace`
- [x] Add git service methods: `Commit`, `Diff`, `Log`, `BranchCreate`, `BranchSwitch`, `BranchList`
- [x] Add rebase service methods: `RebaseRun`, `RebaseStatus`, `RebaseContinue`, `RebaseAbort`
- [x] Add shared service helpers for workspace resolution, default `jobs`, and summary row generation
- [x] Add focused service tests for each new domain method family

## Phase 5C: Module Surface Expansion (`pkg/wsmjs/module`)

- [x] Add manager root `loadWorkspace(name)` returning a workspace handle object
- [x] Add `manager.registry.listWorkspaces()` alias mapped to list workflow
- [x] Add `manager.workspaces.info/add/remove/delete/fork/merge` methods
- [x] Add `manager.git.commit/diff/log` methods
- [x] Add `manager.git.branch.create/switch/list` nested namespace methods
- [x] Add `manager.git.rebase.run/status/continue/abort` nested namespace methods
- [x] Add `WorkspaceHandle` methods (`name`, `path`, `info`, lifecycle methods, `git.*` aliases)
- [x] Ensure flat vs namespaced aliases reuse shared closures to prevent behavior drift
- [x] Add module tests validating export presence, required-field validation, namespace parity, and handle behavior

## Phase 5D: Type Contracts and Docs

- [x] Create `pkg/wsmjs/spec/wsm.d.ts.tmpl` with full completion-level API declarations
- [x] Add generation/validation step for `.d.ts` contract and include in test/lint workflow
- [x] Update `pkg/docs/03-js-api-and-runner.md` with full manager/namespaces/handle method matrix
- [x] Update command reference/troubleshooting docs where JS API examples need expanded coverage
- [x] Add/extend runner/module docs for error semantics and batch row contract behavior

## Phase 5E: Demo Script Expansion (`test/js`)

- [x] Add workspace lifecycle scripts: `08-workspace-info.js`, `09-workspace-add-remove.js`, `10-workspace-delete.js`, `11-workspace-fork-merge.js`
- [x] Add git core scripts: `12-git-commit.js`, `13-git-diff.js`, `14-git-log.js`, `15-git-branch-create-switch-list.js`
- [x] Add rebase scripts: `16-git-rebase-run-happy.js`, `17-git-rebase-status.js`, `18-git-rebase-continue.js`, `19-git-rebase-abort.js`
- [x] Add handle/parity scripts: `20-workspace-handle-basics.js`, `21-workspace-handle-git.js`, `22-flat-vs-namespace-parity-extended.js`
- [x] Update `test/js/README.md` with prerequisites, expected outputs, and script-to-scenario mapping for 00-22

## Phase 5F: Integration Scenario Expansion (`test/integration/scenarios`)

- [x] Add `js_runner_workspace_lifecycle_scenarios_test.go` executing scripts 08-11 in sandbox lifecycle flows
- [x] Add `js_runner_git_ops_scenarios_test.go` executing scripts 12-15 in commit/diff/log/branch flows
- [x] Add `js_runner_rebase_scenarios_test.go` executing scripts 16-19 for happy + conflict flows
- [x] Add `js_runner_workspace_handle_scenarios_test.go` executing scripts 20-22 for handle/parity coverage
- [x] Ensure all JS runner scenarios use data-mode parsing and assert both runner-row and script-result contracts

## Phase 5G: Validation, Delivery, and Handoff

- [x] Run targeted tests for new module/service/unit and JS runner scenario files
- [x] Run full `go test ./test/integration/scenarios` and relevant package suites
- [x] Run `docmgr doctor --ticket WSM-MO-007-JS-API-DESIGN --stale-after 30` and resolve findings
- [x] Update ticket changelog + diary with implementation details, commands, and failures
- [x] Upload refreshed design+plan+diary bundle to reMarkable and verify remote listing
