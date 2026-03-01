---
Title: Workspace Manager JavaScript API Brainstorm and Design Options
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
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/api_sessions.go
      Note: Fluent builder/session API pattern
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/module.go
      Note: Module export organization and hidden ref pattern
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl
      Note: Typed declaration generation strategy
    - Path: ../../../../../../../geppetto/pkg/js/runtimebridge/bridge.go
      Note: Safe JS callback bridge around runtimeowner
    - Path: ../../../../../../../go-go-goja/engine/factory.go
      Note: Runtime factory and composition model
    - Path: ../../../../../../../go-go-goja/modules/common.go
      Note: Native module contract and registry pattern
    - Path: ../../../../../../../go-go-goja/pkg/runtimeowner/runner.go
      Note: Runtime ownership and concurrency guarantees
    - Path: cmd/wsm/cmds/js/root.go
      Note: JS command registration
    - Path: cmd/wsm/cmds/js/runner.go
      Note: Runner CLI verb to execute JS API scripts
    - Path: cmd/wsm/root.go
      Note: |-
        Current CLI composition and grouped command registration
        Root wiring for runner verb
    - Path: demo/js/wsm-api-smoke.js
      Note: Runnable smoke example for new JS API
    - Path: pkg/wsm/branch/types.go
      Note: Typed enums and branch plan DTOs
    - Path: pkg/wsm/gitclient/client.go
      Note: Portable git backend abstraction
    - Path: pkg/wsm/workflows/create_workflow.go
      Note: Request/result workflow contract pattern
    - Path: pkg/wsm/workflows/discover_workflow.go
      Note: Discovery request normalization and orchestration
    - Path: pkg/wsm/workflows/rebase_workflow.go
      Note: High-complexity orchestration and batch result model
    - Path: pkg/wsm/workflows/status_workflow.go
      Note: Status orchestration and workspace detection logic
    - Path: pkg/wsm/workspace.go
      Note: Core workspace lifecycle service surface
    - Path: pkg/wsmjs/module/module.go
      Note: Implemented native require(wsm) module surface
    - Path: pkg/wsmjs/runner/runner.go
      Note: Implemented script runtime helper used by runner verb
    - Path: pkg/wsmjs/service/manager.go
      Note: Implemented workflow facade aligned with recommended architecture
ExternalSources: []
Summary: Evidence-based brainstorm of multiple JavaScript API shapes for Workspace Manager, with recommendation and phased implementation hints.
LastUpdated: 2026-02-28T19:42:00-05:00
WhatFor: Select and implement a JavaScript API surface for workspace-manager that is ergonomic in JS while preserving clean Go package boundaries.
WhenToUse: Use when deciding how to expose workspace-manager workflows to JavaScript and when implementing the first require("wsm") module.
---



# Workspace Manager JavaScript API Brainstorm and Design Options

## Executive Summary

This document proposes and compares several JavaScript API shapes for Workspace Manager (`wsm`) that can run inside goja and potentially in other JS runtimes later.

The current codebase already has strong reusable Go services (`pkg/wsm/*`, `pkg/wsm/workflows/*`) and branch-typing work (`pkg/wsm/branch/types.go`). The best near-term path is to expose these services through a native go-go-goja module (`require("wsm")`) with:

1. A primary object-oriented/session-oriented API for ergonomics and discoverability.
2. A functional API layer for scripting and testability.
3. Typed constants/enums and generated `.d.ts` contracts following geppetto patterns.

Recommended direction: a hybrid API where object handles (`workspace`, `session`, `registry`) are first-class, but all major operations also exist as pure function entrypoints for automation scripts.

## Problem Statement

We need a JavaScript API for Workspace Manager functionality that can be consumed from JS scripts (embedded via goja) while remaining cleanly mapped to the existing `pkg/` architecture.

Requirements inferred from codebase direction and current refactor goals:

1. Keep command-specific presentation logic out of the API.
2. Reuse existing package behavior (`pkg/wsm`, workflows, branch abstractions) rather than re-implementing in JS glue.
3. Support both human-authored scripts and deterministic automation.
4. Keep runtime-thread correctness for JS callback execution.
5. Produce a clear intern-friendly API with explicit contracts and examples.

## Scope

In scope:

1. Analyze relevant architecture in `workspace-manager`, `go-go-goja`, and `geppetto/js`.
2. Compare multiple JS API styles with examples and tradeoffs.
3. Recommend a target API and phased implementation path.
4. Provide implementation hints (module wiring, DTO codecs, error model, typing strategy).

Out of scope:

1. Full implementation of the JS module.
2. Backward compatibility with old command-only interfaces.
3. External npm packaging strategy.

## Evidence-Based Current Architecture

## Workspace manager package surfaces

Observed:

1. CLI entry wires grouped command registrars (`registry`, `workspace`, `git`) from `cmd/wsm/root.go` lines 52-74.
2. Workspace lifecycle logic is centralized in `pkg/wsm/workspace.go` with a broad `WorkspaceManager` type (`NewWorkspaceManager`, `CreateWorkspace`, `DeleteWorkspace`, `AddRepositoryToWorkspace`, etc.; lines 21-1063+).
3. A workflow/request-result style already exists for many operations:
   - `CreateRequest` / `CreateResult` and `Create(ctx, req)` in `pkg/wsm/workflows/create_workflow.go` lines 12-61.
   - `DiscoverRequest` / `DiscoverResult` in `pkg/wsm/workflows/discover_workflow.go` lines 14-53.
   - `StatusRequest` with `GetStatus` in `pkg/wsm/workflows/status_workflow.go` lines 11-57.
   - `RebaseRequest` / `RebaseResult` and action/status flows in `pkg/wsm/workflows/rebase_workflow.go` lines 29-228.
4. Git backend abstraction is already explicit in `pkg/wsm/gitclient/client.go` lines 47-88.
5. Branch refactor introduced typed domain values and enums in `pkg/wsm/branch/types.go` lines 3-125 (`BranchName`, `RemoteName`, `ResolutionMode`, `ResolutionStrategy`, `RemoteRefKind`).

Implication:

The API should not expose raw CLI semantics first. It should expose typed workflow/service capabilities already present in `pkg/`.

## go-go-goja architecture patterns

Observed:

1. Native module contract is minimal and stable: `Name()`, `Doc()`, `Loader(*goja.Runtime, *goja.Object)` (`go-go-goja/modules/common.go` lines 30-34).
2. Central module registry/enable path exists (`EnableAll`) (`go-go-goja/modules/common.go` lines 98-103).
3. Runtime composition has moved toward an immutable builder/factory model (`go-go-goja/engine/factory.go` lines 15-132).
4. Runtime ownership is explicit and safe via `runtimeowner.Runner` (`Call`, `Post`) with cancellation and panic handling (`go-go-goja/pkg/runtimeowner/runner.go` lines 62-178).

Implication:

A `wsm` JS module should:

1. Be a standard native module (`require("wsm")`).
2. Avoid direct unsynchronized goja runtime mutation from goroutines.
3. Use runtimeowner-friendly patterns for callbacks/events.

## geppetto/js patterns worth reusing

Observed:

1. Module registration pattern with option struct + `Register(reg, opts)` in `geppetto/pkg/js/modules/geppetto/module.go` lines 32-55.
2. Rich export tree setup in `installExports` (lines 137-190), grouping related surfaces (`turns`, `engines`, `profiles`, etc.).
3. Builder/session ergonomics (`createBuilder`, `createSession`, `runInference`) with fluent methods in `api_sessions.go` lines 18-205.
4. Hidden Go references attached to JS objects (`__geppetto_ref`) in `module.go` lines 198-221.
5. Runtime bridge wrapper around `runtimeowner.Runner` (`geppetto/pkg/js/runtimebridge/bridge.go` lines 11-84).
6. Typed `.d.ts` generation strategy (template-based) in `geppetto.d.ts.tmpl` lines 3+.

Implication:

For `wsm`, geppetto is a strong reference for:

1. Module options injection.
2. Export grouping.
3. Handle-based object APIs.
4. Type declaration generation.

## Design Goals

1. Ergonomic JS usage for common flows (`discover -> create -> status`, `rebase status/continue/abort`).
2. Deterministic and explicit behavior for automation (no hidden interactive logic).
3. Clean mapping to existing Go services/workflows.
4. Clear and typed API contracts (TS-friendly even in goja-only runtime).
5. Future extension path for async/evented operations.

## API Style Options (Brainstorm)

## Option A: Pure Functional API

Shape:

```javascript
const wsm = require("wsm");

const discovered = wsm.discover({ paths: ["~/code"], recursive: true, maxDepth: 3 });
const created = wsm.createWorkspace({
  name: "feature-js-api",
  repos: ["workspace-manager", "geppetto"],
  branch: "feature/js-api",
  baseBranch: "main"
});
const status = wsm.getStatus({ workspaceName: created.workspace.name, jobs: 8 });
```

Pros:

1. Easy to test and mock.
2. Lowest conceptual overhead.
3. Straightforward mapping from workflow request/result DTOs.

Cons:

1. No stateful handles for multi-step operations.
2. Harder to attach shared options/context incrementally.
3. Can become a flat, crowded namespace.

Fit with current code:

Very high for workflows; medium for advanced orchestration and incremental configuration.

## Option B: Object/Handle API (Manager + Workspace handles)

Shape:

```javascript
const wsm = require("wsm");

const manager = wsm.createManager({
  defaultJobs: 8,
  defaultRemote: wsm.consts.remote.ORIGIN,
});

const workspace = manager.createWorkspace({
  name: "feature-js-api",
  repos: ["workspace-manager", "go-go-goja"],
  branch: "feature/js-api",
  baseBranch: "main",
});

const status = workspace.status();
workspace.rebase({ targetBranch: "main", jobs: 4 });
```

Pros:

1. Better discoverability and onboarding.
2. Natural place for shared defaults and caches.
3. Aligns with geppetto handle pattern (`attachRef`, hidden Go refs).

Cons:

1. Slightly more adapter complexity.
2. Need careful lifecycle semantics for handles.

Fit with current code:

High. Maps to `WorkspaceManager`, context services, and workflow objects.

## Option C: Builder + Session API (geppetto-like)

Shape:

```javascript
const wsm = require("wsm");

const session = wsm
  .createSessionBuilder()
  .withWorkspace("feature-js-api")
  .withJobs(12)
  .withRepoFilter(["workspace-manager", "geppetto"])
  .build();

const rebasePlan = session.planRebase({ targetBranch: "main" });
const run = session.runRebase(rebasePlan);
```

Pros:

1. Strong when operations need optional knobs.
2. Good for complex multi-step operations (plan/apply/rollback).
3. Familiar if team already uses geppetto JS APIs.

Cons:

1. Too heavy for simple one-off scripts.
2. Requires more API surface and docs.

Fit with current code:

Medium-high; best for rebase/merge-like complex workflows.

## Option D: Command-wrapper API (thin over CLI verbs)

Shape:

```javascript
const wsm = require("wsm");
wsm.run("create", { name: "x", repos: ["a", "b"] });
wsm.run("status", { workspace: "x" });
```

Pros:

1. Fastest bootstrap.
2. Mirrors CLI knowledge directly.

Cons:

1. Leaks command-level technical debt into API.
2. Weak typing and weak discoverability.
3. Hard to evolve cleanly.

Fit with current code:

Low for long-term architecture quality.

## Option E: Resource-centric sub-APIs (`wsm.workspaces`, `wsm.registry`, `wsm.git`)

Shape:

```javascript
const wsm = require("wsm");

const ws = wsm.workspaces.create({ name: "feat-1", repos: ["workspace-manager"] });
const repos = wsm.registry.listRepositories();
const rebaseRows = wsm.git.rebase({ workspace: ws.name, targetBranch: "main", jobs: 4 });
```

Pros:

1. Good namespace clarity.
2. Easier to keep module organized as it grows.
3. Matches grouped CLI layout (registry/workspace/git) without inheriting CLI coupling.

Cons:

1. Requires curation to avoid duplication between groups.
2. Slightly more verbose than flat API.

Fit with current code:

High and scalable.

## Recommended API Direction

Recommendation: combine Option B + Option E + selective Option A.

Target shape:

1. `wsm.createManager(opts)` for stateful defaults.
2. `wsm.workspaces.*`, `wsm.registry.*`, `wsm.git.*` grouped functional helpers.
3. `manager.*` methods mirror grouped helpers and can return object handles.
4. A few advanced builder/session-style APIs only for complex workflows (rebase, merge) later.

Why this is strongest now:

1. Preserves clean `pkg/` architecture and service boundaries.
2. Easy for interns: grouped APIs + explicit manager object.
3. Keeps low-friction scripting entrypoints available.
4. Avoids freezing API into command-wrapper semantics.

## Proposed Public API Sketch (v1 brainstorm)

```typescript
declare module "wsm" {
  export const version: string;

  export const consts: {
    resolutionMode: {
      readonly CREATE_WORKTREE: "create-worktree";
      readonly ADD_REPOSITORY: "add-repository";
      readonly SYNC: "sync";
    };
    resolutionStrategy: {
      readonly USE_LOCAL: "use-local";
      readonly TRACK_REMOTE: "track-remote";
      readonly CREATE_FROM_BASE: "create-from-base";
      readonly CREATE_FROM_HEAD: "create-from-head";
    };
    remote: {
      readonly ORIGIN: "origin";
    };
  };

  export interface ManagerOptions {
    defaultJobs?: number;
    defaultRemote?: string;
    workspaceDir?: string;
  }

  export interface CreateWorkspaceInput {
    name: string;
    repos: string[];
    branch?: string;
    branchPrefix?: string;
    baseBranch?: string;
    agentSource?: string;
    dryRun?: boolean;
  }

  export interface WorkspaceHandle {
    name(): string;
    path(): string;
    repositories(): string[];
    status(opts?: { jobs?: number }): WorkspaceStatus;
    rebase(opts: { targetBranch: string; repository?: string; interactive?: boolean; dryRun?: boolean; jobs?: number }): RebaseResult[];
    delete(opts?: { removeFiles?: boolean; forceWorktrees?: boolean }): void;
  }

  export interface Manager {
    discover(input: { paths?: string[]; recursive?: boolean; maxDepth?: number }): DiscoverResult;
    createWorkspace(input: CreateWorkspaceInput): CreateWorkspaceResult;
    loadWorkspace(name: string): WorkspaceHandle;
    listWorkspaces(): WorkspaceSummary[];

    // Thin grouped namespaces (functional style)
    registry: {
      listRepositories(): RepositorySummary[];
    };
    workspaces: {
      info(name: string): WorkspaceInfo;
      removeRepository(input: { workspaceName: string; repoName: string; force?: boolean; removeFiles?: boolean }): void;
      addRepository(input: { workspaceName: string; repoName: string; branchName?: string; forceOverwrite?: boolean }): void;
    };
    git: {
      status(input: { workspaceName?: string; jobs?: number }): WorkspaceStatus;
      commit(input: CommitRequest): CommitResult;
      rebase(input: RebaseRequest): RebaseResult[];
      rebaseStatus(input: { workspaceName: string; repository?: string; jobs?: number }): RebaseStatusRow[];
      rebaseContinue(input: { workspaceName: string; repository?: string; jobs?: number }): RebaseActionRow[];
      rebaseAbort(input: { workspaceName: string; repository?: string; jobs?: number }): RebaseActionRow[];
    };
  }

  export function createManager(opts?: ManagerOptions): Manager;

  // Functional convenience wrappers
  export function discover(input: { paths?: string[]; recursive?: boolean; maxDepth?: number }): DiscoverResult;
  export function createWorkspace(input: CreateWorkspaceInput): CreateWorkspaceResult;
  export function status(input?: { workspaceName?: string; jobs?: number }): WorkspaceStatus;
}
```

## Error Model Options

## Error Model 1: Throw on failure (recommended default)

Behavior:

1. Validation and operational failures throw JS errors (`panic(vm.NewGoError(err))` in loader methods).
2. Results are clean on success.

Pros:

1. Idiomatic for goja native modules and geppetto patterns.
2. Simple happy-path API.

Cons:

1. Batch operations may need partial result semantics.

## Error Model 2: Never throw, always return `{ok,error}` envelopes

Pros:

1. Deterministic pipeline style.

Cons:

1. Noisy and unidiomatic for JS consumers.
2. Encourages missed error checks.

## Recommended compromise

1. Throw on function-level failures.
2. Keep structured per-repo statuses for bulk operations (`RebaseResult.Success/Error`, etc.).
3. Add explicit `strict` option later for all-or-nothing semantics where needed.

## Runtime and Concurrency Notes

Constraints:

1. goja runtime is single-threaded by ownership model.
2. Any callback/event interop must pass through a runner/bridge pattern.

Guidance:

1. Do not call JS callables from background goroutines directly.
2. Mirror geppetto runtime bridge style for future event hooks.
3. Keep heavy git operations in Go; only marshal final DTOs to JS.

Pseudo-flow:

```go
// In module loader
func (m *module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    rt := newRuntime(vm, m.opts) // includes runner + services
    exports := moduleObj.Get("exports").(*goja.Object)
    rt.installExports(exports)
}

func (rt *runtime) createManager(call goja.FunctionCall) goja.Value {
    opts, err := decodeManagerOptions(call.Argument(0))
    if err != nil { panic(rt.vm.NewGoError(err)) }

    mgr, err := newManagerFacade(rt, opts) // wraps pkg/wsm workflows/services
    if err != nil { panic(rt.vm.NewGoError(err)) }

    obj := rt.vm.NewObject()
    rt.attachRef(obj, mgr) // hidden Go ref
    rt.installManagerMethods(obj, mgr)
    return obj
}
```

## Mapping to Existing Go Packages

## Suggested internal layering

1. `pkg/wsmjs/service` (new): pure-Go façade interfaces and DTO adapters over `pkg/wsm` and `pkg/wsm/workflows`.
2. `pkg/wsmjs/module` (new): goja-native module adapter (`require("wsm")`), decoding/encoding, exports.
3. `pkg/wsmjs/spec` (new): `.d.ts` templates and generated TS declarations.

Proposed Go interfaces (internal):

```go
type WorkspaceAPI interface {
    Discover(ctx context.Context, req workflows.DiscoverRequest) (*workflows.DiscoverResult, error)
    Create(ctx context.Context, req workflows.CreateRequest) (*workflows.CreateResult, error)
    Status(ctx context.Context, req workflows.StatusRequest) (*wsm.WorkspaceStatus, error)
    Rebase(ctx context.Context, workspaceName string, req workflows.RebaseRequest) ([]workflows.RebaseResult, error)
}
```

Rationale:

1. Keeps `module` package thin and codec-focused.
2. Allows unit testing service layer without goja runtime.
3. Makes future HTTP/MCP/API adapters easy.

## API Grouping Diagram

```text
require("wsm")
  |
  +-- createManager(opts) -> ManagerHandle
  |      |
  |      +-- discover/create/load/list
  |      +-- registry.*
  |      +-- workspaces.*
  |      +-- git.*
  |
  +-- functional shortcuts
  |      +-- discover/createWorkspace/status
  |
  +-- consts (typed enums and literals)
```

## Data and Enum Strategy

Use typed enum constants sourced from Go domain types where possible.

Candidate mappings:

1. `branch.ResolutionMode` -> `wsm.consts.resolutionMode.*`
2. `branch.ResolutionStrategy` -> `wsm.consts.resolutionStrategy.*`
3. `branch.RemoteRefKind` -> `wsm.consts.remoteRefKind.*`
4. `branch.DefaultRemoteName` -> `wsm.consts.remote.ORIGIN`

Implementation hint:

1. Add explicit encode/decode helpers in module code.
2. Avoid raw string switches sprinkled across command adapters.

## Implementation Plan (Phased)

## Phase 0: Contract-first design lock

1. Finalize JS API surface and names.
2. Freeze DTO shapes for v1.
3. Decide final error semantics.

Deliverables:

1. API contract markdown and `.d.ts` template draft.
2. Symbol inventory for grouped exports.

## Phase 1: Internal Go façade (`pkg/wsmjs/service`)

1. Build service wrapper over existing workflows/services.
2. Add normalization for workspace lookup, defaults, and job counts.
3. Add unit tests around façade behavior.

Deliverables:

1. `pkg/wsmjs/service/*.go`
2. unit tests for create/discover/status/rebase actions.

## Phase 2: Native module adapter (`pkg/wsmjs/module`)

1. Implement module registration and loader.
2. Install grouped exports (`registry`, `workspaces`, `git`, `consts`).
3. Implement manager and workspace handle objects with hidden refs.
4. Implement functional shortcuts.

Deliverables:

1. `pkg/wsmjs/module/module.go`
2. `pkg/wsmjs/module/api_*.go`
3. runtime-level integration tests (`require("wsm")`).

## Phase 3: Typings and docs

1. Create `spec/wsm.d.ts.tmpl` and generation command.
2. Add usage examples and migration notes.
3. Ensure help/README examples match real signatures.

Deliverables:

1. generated `.d.ts`
2. examples in `cmd/examples` or ticket docs.

## Phase 4: Advanced surfaces (optional)

1. Add builder/session API for complex orchestrations if needed.
2. Add event hooks and callback bridge if async reporting becomes required.

## Testing and Validation Strategy

1. Service-level unit tests (no goja).
2. Runtime integration tests that boot go-go-goja runtime and execute JS requiring `wsm`.
3. Golden tests for DTO marshaling where field naming is sensitive.
4. Regression tests for typed constants/enums.

Test examples:

```bash
go test ./pkg/wsmjs/service -count=1
go test ./pkg/wsmjs/module -count=1
GOWORK=off go test ./... -count=1
```

Acceptance criteria for v1:

1. `require("wsm")` loads in runtime.
2. `createManager().discover/createWorkspace/status` works against fixture repos.
3. `git.rebaseStatus/rebaseContinue/rebaseAbort` wrappers expose current workflow capability.
4. `.d.ts` file describes public API correctly.

## Risks and Mitigations

1. Risk: API drift from underlying workflows.
   Mitigation: service façade owns adapters and tests assert mapping.

2. Risk: runtime-thread misuse in callbacks.
   Mitigation: if callbacks are introduced, require runtimebridge/runner path.

3. Risk: overexposure of unstable commands.
   Mitigation: expose only pkg-backed behavior; no direct command wrapper API.

4. Risk: large `workspace.go` surface complexity leaks into JS.
   Mitigation: gate through focused façade methods instead of direct pass-through.

## Alternatives Considered

1. CLI-shell wrapper API only (`wsm.run("verb", ...)`): rejected for long-term cleanliness and typing.
2. Builder-only API: rejected as too heavy for common one-off scripts.
3. Functional-only flat API: acceptable but weaker discoverability at scale.

## Open Questions

1. Should v1 expose mutating operations like `deleteWorkspace` directly, or keep read-first + create-first for safety?
2. Do we need async promise-like wrappers in goja v1, or keep everything synchronous initially?
3. Should profile/setup-script operations from `workspace.go` be publicly exposed in JS v1 or deferred?
4. What is the expected host process lifecycle for long-running workspace sessions?

## Recommended Decision

Adopt hybrid Manager + grouped namespaces with functional shortcuts now, and defer advanced builder/session/event APIs until real demand appears.

This gives a clean and scalable API that matches current package architecture and can be implemented without command-level baggage.

## References

Primary evidence files:

1. [workspace-manager/cmd/wsm/root.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/root.go)
2. [workspace-manager/pkg/wsm/workspace.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workspace.go)
3. [workspace-manager/pkg/wsm/workflows/create_workflow.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workflows/create_workflow.go)
4. [workspace-manager/pkg/wsm/workflows/discover_workflow.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workflows/discover_workflow.go)
5. [workspace-manager/pkg/wsm/workflows/status_workflow.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workflows/status_workflow.go)
6. [workspace-manager/pkg/wsm/workflows/rebase_workflow.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workflows/rebase_workflow.go)
7. [workspace-manager/pkg/wsm/branch/types.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/branch/types.go)
8. [workspace-manager/pkg/wsm/gitclient/client.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/client.go)
9. [go-go-goja/modules/common.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/go-go-goja/modules/common.go)
10. [go-go-goja/engine/factory.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/go-go-goja/engine/factory.go)
11. [go-go-goja/pkg/runtimeowner/runner.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/go-go-goja/pkg/runtimeowner/runner.go)
12. [geppetto/pkg/js/modules/geppetto/module.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/geppetto/pkg/js/modules/geppetto/module.go)
13. [geppetto/pkg/js/modules/geppetto/api_sessions.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/geppetto/pkg/js/modules/geppetto/api_sessions.go)
14. [geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl)
15. [geppetto/pkg/js/runtimebridge/bridge.go](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/geppetto/pkg/js/runtimebridge/bridge.go)
