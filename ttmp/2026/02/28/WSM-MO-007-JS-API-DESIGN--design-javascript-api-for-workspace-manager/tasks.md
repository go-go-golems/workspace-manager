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
