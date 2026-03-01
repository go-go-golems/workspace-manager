---
Title: WSM JS API and Runner
Slug: wsm-js-api-and-runner
Short: Use `wsm runner` and `require("wsm")` for scriptable workspace automation.
Topics:
- workspace-manager
- javascript
- api-design
Commands:
- runner
Flags:
- --print-result
- --output-mode
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

WSM includes an embedded JavaScript runtime (powered by goja, a pure-Go
ECMAScript 5.1 engine). This lets you write automation scripts that call WSM
operations directly, without shelling out to the CLI.

This is useful when you need to chain multiple workspace operations together,
query repository state programmatically, or build custom tooling on top of WSM.

## Running a script

Use `wsm runner` to execute a JavaScript file:

```bash
wsm runner my-script.js
```

The script's final expression becomes its return value. By default, WSM prints
this value as formatted output. To suppress it:

```bash
wsm runner my-script.js --print-result=false
```

For machine-readable output:

```bash
wsm runner my-script.js --output-mode data
```

## The `require("wsm")` module

Inside `wsm runner`, the `wsm` module is pre-registered and available via
`require("wsm")`. You do not need to install anything -- the module is built
into the runner.

```javascript
const wsm = require("wsm");
```

### Top-level exports

| Export | Type | Description |
|--------|------|-------------|
| `wsm.version` | string | Module version (currently `"0.1.0"`) |
| `wsm.consts` | object | Branch resolution and remote constants |
| `wsm.createManager(options)` | function | Create a Manager handle |
| `wsm.discover(input)` | function | Discover repositories (convenience) |
| `wsm.createWorkspace(input)` | function | Create a workspace (convenience) |
| `wsm.status(input)` | function | Get workspace status (convenience) |

The convenience functions (`discover`, `createWorkspace`, `status`) create a
temporary manager under the hood. For scripts that make multiple calls, create a
manager once and reuse it.

### Creating a Manager

```javascript
const manager = wsm.createManager({ defaultJobs: 8 });
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `defaultJobs` | number | `8` | Default parallelism for operations |

### Manager methods

The manager object exposes operations both as flat methods and as grouped
namespaces:

**Flat methods** (direct on the manager):

| Method | Description |
|--------|-------------|
| `manager.discover(input)` | Discover repositories |
| `manager.createWorkspace(input)` | Create a new workspace |
| `manager.status(input)` | Get workspace status |
| `manager.listWorkspaces()` | List all workspaces |
| `manager.listRepositories(input)` | List discovered repositories |

**Grouped namespaces** (alternative access pattern):

| Namespace | Method | Same as |
|-----------|--------|---------|
| `manager.registry` | `.listRepositories(input)` | `manager.listRepositories` |
| `manager.workspaces` | `.create(input)` | `manager.createWorkspace` |
| `manager.workspaces` | `.list()` | `manager.listWorkspaces` |
| `manager.workspaces` | `.status(input)` | `manager.status` |
| `manager.git` | `.status(input)` | `manager.status` |

Both access patterns call the same underlying implementation. Use whichever
reads better in your script.

### Input types

**DiscoverInput**:

```javascript
manager.discover({
  paths: ["/home/user/code"],
  recursive: true,    // default: true
  maxDepth: 3,        // default: 3
});
```

**CreateWorkspaceInput**:

```javascript
manager.createWorkspace({
  name: "my-feature",       // required
  repos: ["wsm", "geppetto"], // required
  branch: "",               // optional, auto-generated if empty
  branchPrefix: "task",     // default prefix for auto-generation
  baseBranch: "",           // optional, defaults to repo default
  agentSource: "",          // optional, AGENT.md template path
  dryRun: false,            // preview without creating
});
```

**StatusInput**:

```javascript
manager.status({
  workspaceName: "my-feature",
  jobs: 4,                  // defaults to manager's defaultJobs
});
```

**ListRepositoriesInput**:

```javascript
manager.listRepositories({
  tags: ["go", "cli"],      // optional filter
});
```

### Constants

The `wsm.consts` object exposes branch resolution enums and remote defaults.
These match the Go types in `pkg/wsm/branch/`.

```javascript
wsm.consts.resolutionMode.CREATE_WORKTREE
wsm.consts.resolutionMode.ADD_REPOSITORY
wsm.consts.resolutionMode.SYNC

wsm.consts.resolutionStrategy.USE_LOCAL
wsm.consts.resolutionStrategy.TRACK_REMOTE
wsm.consts.resolutionStrategy.CREATE_FROM_BASE
wsm.consts.resolutionStrategy.CREATE_FROM_HEAD

wsm.consts.remoteRefKind.NONE
wsm.consts.remoteRefKind.REMOTE_TRACKING_BRANCH

wsm.consts.remote.ORIGIN  // "origin"
```

## Complete example

This script discovers repositories, lists them, and returns a summary:

```javascript
const wsm = require("wsm");

var manager = wsm.createManager({ defaultJobs: 4 });

// List all discovered repositories
var repos = manager.registry.listRepositories({ tags: [] });

// List all workspaces
var workspaces = manager.listWorkspaces();

({
  version: wsm.version,
  repositoryCount: repos.length,
  workspaceCount: workspaces.length,
  defaultRemote: wsm.consts.remote.ORIGIN,
});
```

Run it:

```bash
wsm runner summary.js
```

## Error handling

API methods throw on error. In goja, thrown Go errors surface as JavaScript
exceptions. A script that calls an operation on a non-existent workspace will
terminate with an error message.

For bulk operations (like status across multiple repos), the result object
contains per-repository success/failure information rather than throwing on the
first failure.

## Design notes

- The JS API routes through the same Go workflow/service layer as the CLI.
  There are no separate code paths -- `manager.createWorkspace()` calls the
  same `CreateWorkflow` that `wsm create` uses.
- Result values are converted to plain JavaScript objects via JSON round-trip
  for predictable types. Go structs with `time.Time` fields become ISO strings,
  slices become arrays, etc.
- The goja runtime is single-threaded. All operations run synchronously.
  Parallelism within an operation (e.g. `--jobs`) is handled in Go, not JS.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `Cannot find module 'wsm'` | Script not run through the runner | Use `wsm runner <script.js>`, not `node` |
| Empty or missing result | Script ends with a statement, not an expression | End the script with an object literal: `({ key: value })` |
| Unexpected data in automations | Human output mode enabled | Use `--output-mode data --print-result=false` |
| `createWorkspace requires name` | Missing required field | Pass `{ name: "...", repos: [...] }` |

## See Also

- `wsm help wsm-command-reference`
- `wsm help wsm-architecture-overview`
- `wsm help wsm-troubleshooting`
