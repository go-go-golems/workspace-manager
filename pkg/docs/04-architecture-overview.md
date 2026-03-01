---
Title: WSM Architecture Overview
Slug: wsm-architecture-overview
Short: High-level architecture map for commands, workflows, services, and JS integration.
Topics:
- workspace-manager
- architecture
- refactor
Commands:
- all
Flags:
- --output-mode
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page describes how the WSM codebase is organized. It is aimed at
contributors who want to understand where behavior lives, how layers connect,
and where to add new functionality.

## Layering model

WSM follows a strict four-layer architecture. Each layer has a clear
responsibility, and dependencies flow downward:

```
┌──────────────────────────────────────────────────┐
│  cmd/wsm/cmds/          CLI adapters             │
│    registry/   workspace/   git/   js/           │
├──────────────────────────────────────────────────┤
│  pkg/wsm/workflows/     Orchestration layer      │
│    create, status, commit, rebase, fork, ...     │
├──────────────────────────────────────────────────┤
│  pkg/wsm/               Core domain services     │
│    workspace, discovery, git_operations, branch/ │
├──────────────────────────────────────────────────┤
│  pkg/wsm/gitclient/     Git abstraction          │
│    hybrid (pure-Go + CLI fallback)               │
└──────────────────────────────────────────────────┘
```

### Layer 1: CLI adapters (`cmd/wsm/cmds/`)

Each Cobra command lives in a subpackage. Command files are thin: they decode
flags into a settings struct, call a workflow, and format output. Business logic
does not belong here.

Subpackages:
- `registry/` -- `discover`, `list repos`, `list workspaces`
- `workspace/` -- `create`, `info`, `status`, `add`, `remove`, `fork`, `merge`, `delete`
- `git/` -- `commit`, `diff`, `log`, `branch` (create/switch/list), `rebase` (start/status/continue/abort)
- `js/` -- `runner`

Shared infrastructure lives in `cmd/wsm/cmds/common/`:
- `build.go` -- Glazed command description builder with standard sections
- `runtime.go` -- `--output-mode` resolution and structured row emission

### Layer 2: Workflows (`pkg/wsm/workflows/`)

Workflows are the orchestration layer. Each workflow encapsulates a complete
user-facing operation and defines its own request/result types:

| Workflow | Purpose |
|----------|---------|
| `DiscoverWorkflow` | Scan directories, update registry |
| `CreateWorkflow` | Resolve branches, create worktrees, write workspace config |
| `StatusWorkflow` | Gather git status across repos (parallelized with `--jobs`) |
| `CommitWorkflow` | Two-phase: `Prepare()` then `Execute()` for multi-repo commits |
| `RebaseWorkflow` | Rebase/status/continue/abort with conflict detection |
| `ListWorkflow` | List repositories and workspaces |
| `InfoWorkflow` | Resolve workspace, extract field values |
| `ForkWorkflow` | Two-phase: `Plan()` then `Fork()` |
| `DeleteWorkflow` | Preview then delete with worktree cleanup |
| `MergeWorkflow` | Merge branches back and optionally clean up |

Workflows own the sequencing of operations. They call into core domain services
but never call CLI commands.

### Layer 3: Core domain (`pkg/wsm/`)

The domain layer contains the foundational types and services:

- **`types.go`** -- `Repository`, `Workspace`, `RepositoryStatus`, `WorkspaceStatus`, `WorkspaceConfig`
- **`workspace.go`** -- `WorkspaceManager` with create, load, delete, detect operations
- **`workspace_context.go`** -- Workspace detection from working directory
- **`discovery.go`** -- Repository scanning logic
- **`status.go`** -- `StatusChecker` for per-repo git status
- **`git_operations.go`** -- Diff and other git operations
- **`history_operations.go`** -- Log and commit history queries
- **`branch_operations.go`** -- Create, switch, list branches across repos
- **`rebase_operations.go`** -- Low-level rebase: start, continue, abort, detect state, list conflicts

### Layer 4: Git client (`pkg/wsm/gitclient/`)

The git client provides an abstraction over git operations. WSM uses a hybrid
approach:

- **`client.go`** -- Interface definition
- **`gogit_client.go`** -- Pure-Go implementation via go-git (fast, no shell)
- **`cli_client.go`** -- Falls back to `git` CLI for operations go-git cannot handle
- **`hybrid_client.go`** -- Selects the right backend per operation
- **`worktree_cli.go`** -- Worktree operations always go through CLI (go-git lacks worktree support)

## Branch resolution system

Branch policy is centralized in `pkg/wsm/branch/` to avoid duplicating branch
decisions across commands. The system uses typed enums:

**ResolutionMode** -- what operation triggered the branch decision:
- `CreateWorktree` -- creating a new workspace worktree
- `AddRepository` -- adding a repo to an existing workspace
- `Sync` -- syncing branches across a workspace

**ResolutionStrategy** -- how to obtain the target branch:
- `UseLocal` -- use the existing local branch as-is
- `TrackRemote` -- create a local tracking branch from remote
- `CreateFromBase` -- create a new branch from the specified base
- `CreateFromHead` -- create a new branch from HEAD

**RemoteRefKind** -- how to interpret remote references:
- `None` -- no remote reference
- `RemoteTrackingBranch` -- reference is `remote/branch` format

A `BranchResolutionRequest` goes in, a `BranchResolutionPlan` comes out. The
plan is deterministic: same inputs always produce the same strategy. Commands
and workflows use the branch service rather than implementing their own logic.

## JS integration

The JavaScript layer lives in `pkg/wsmjs/` and mirrors the CLI/workflow
architecture:

```
pkg/wsmjs/
├── service/    Manager facade wrapping workflows
├── module/     goja native module (require("wsm"))
└── runner/     Script file loader and executor
```

- **`service/manager.go`** -- The `Manager` struct wraps workflow instances
  and provides a flat Go API: `Discover`, `CreateWorkspace`, `Status`,
  `ListWorkspaces`, `ListRepositories`.
- **`module/module.go`** -- Registers a goja native module that maps JS calls
  to service methods. Handles type conversion between Go and JS via JSON
  round-trip.
- **`runner/runner.go`** -- `RunFile()` bootstraps a goja runtime, registers
  the `wsm` module, loads and executes a script file, and returns its result.

The key design decision is that the JS API calls the same workflow layer as the
CLI. There are no separate implementations -- `require("wsm").createWorkspace()`
and `wsm create` both go through `CreateWorkflow`.

## Output model

All commands use the shared output mode system defined in
`cmd/wsm/cmds/common/runtime.go`:

- `ShouldOutputHuman(mode)` -- true for `human` and `both` modes
- `ShouldOutputData(mode)` -- true for `data` and `both` modes
- `EmitRows(ctx, vals, rows)` -- emits structured data through Glazed

Commands typically have two output blocks: one for human-readable text (headers,
tables, status messages) and one for structured rows. The `both` mode runs them
sequentially.

## Adding a new command

1. Create the settings struct and command in the appropriate `cmd/wsm/cmds/` subpackage.
2. Create a workflow in `pkg/wsm/workflows/` with request/result types.
3. The workflow should call core services from `pkg/wsm/`, not implement git logic directly.
4. Wire the command into `cmd/wsm/root.go`.
5. If the operation should be available from JS, add a method to `pkg/wsmjs/service/manager.go` and expose it in `pkg/wsmjs/module/module.go`.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| New command has lots of business logic | Logic added in `cmd/` instead of workflow | Move orchestration into `pkg/wsm/workflows/` |
| Duplicate branch decisions | Branch policy bypassed | Route through `pkg/wsm/branch/` service |
| JS API diverges from CLI behavior | Separate code paths | Both must call the same workflow |
| Rebase state detection fails | Missing `.git/rebase-merge` check | See `rebase_operations.go` for state detection logic |

## See Also

- `wsm help wsm-command-reference`
- `wsm help wsm-persistence-and-state`
- `wsm help wsm-js-api-and-runner`
