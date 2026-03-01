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

WSM includes an embedded JavaScript runtime (goja). Scripts can call workspace
operations through `require("wsm")` without shelling out to CLI subcommands.

## Running Scripts

```bash
wsm runner my-script.js
```

Control result printing:

```bash
wsm runner my-script.js --print-result=false
```

Use machine-oriented rows:

```bash
wsm runner my-script.js --output-mode data --output json --print-result=false
```

## Module Overview

Inside runner scripts:

```javascript
const wsm = require("wsm");
const manager = wsm.createManager({ defaultJobs: 8 });
```

Top-level exports:

| Export | Type | Notes |
|---|---|---|
| `wsm.version` | string | Current module version (`"0.2.0"`) |
| `wsm.consts` | object | Branch/resolution constants |
| `wsm.createManager(options)` | function | Returns a manager object |
| `wsm.discover(input)` | function | Convenience alias |
| `wsm.createWorkspace(input)` | function | Convenience alias |
| `wsm.status(input)` | function | Convenience alias |

Manager options:

| Option | Type | Default | Notes |
|---|---|---|---|
| `defaultJobs` | number | `8` | Used when per-call `jobs` is omitted |

## Manager API Surface

Flat manager methods:

| Method | Purpose |
|---|---|
| `manager.discover(input)` | Repository discovery |
| `manager.createWorkspace(input)` | Workspace creation |
| `manager.status(input)` | Workspace status |
| `manager.listWorkspaces()` | Workspace registry listing |
| `manager.listRepositories(input)` | Repository registry listing |
| `manager.loadWorkspace(name)` | Returns workspace handle |
| `manager.info(input)` | Workspace metadata |
| `manager.addRepository(input)` | Add repo to workspace |
| `manager.removeRepository(input)` | Remove repo from workspace |
| `manager.deleteWorkspace(input)` | Delete workspace |
| `manager.forkWorkspace(input)` | Fork workspace |
| `manager.mergeWorkspace(input)` | Merge workspace |
| `manager.commit(input)` | Commit operation |
| `manager.diff(input)` | Diff operation |
| `manager.log(input)` | Log operation |

Namespaced aliases:

| Namespace | Method |
|---|---|
| `manager.registry` | `listRepositories`, `listWorkspaces` |
| `manager.workspaces` | `create`, `list`, `status`, `info`, `add`, `remove`, `delete`, `fork`, `merge` |
| `manager.git` | `status`, `commit`, `diff`, `log` |
| `manager.git.branch` | `create`, `switch`, `list` |
| `manager.git.rebase` | `run`, `status`, `continue`, `abort` |

Flat and namespaced methods route to shared closures in module code, so behavior
stays aligned.

## Workspace Handle API

Load and operate in a workspace-scoped context:

```javascript
const ws = manager.loadWorkspace("my-workspace");
const info = ws.info();
const log = ws.git.log({ limit: 20, oneline: true });
```

Handle methods:

| Method | Purpose |
|---|---|
| `ws.name()` | Workspace name |
| `ws.path()` | Workspace path |
| `ws.info(input?)` | Workspace info |
| `ws.status(input?)` | Workspace status |
| `ws.addRepository(input)` | Add repo |
| `ws.removeRepository(input)` | Remove repo |
| `ws.delete(input?)` | Delete this workspace |
| `ws.merge(input?)` | Merge this workspace |
| `ws.git.*` | Scoped git/branch/rebase methods |

## Common Inputs

Examples (camelCase keys):

```javascript
manager.discover({ paths: ["/work/repos"], recursive: true, maxDepth: 3 });

manager.createWorkspace({
  name: "ws-api-demo",
  repos: ["repo1", "repo2"],
  branchPrefix: "feat",
  dryRun: false,
});

manager.workspaces.add({
  workspaceName: "ws-api-demo",
  repoName: "repo3",
  force: true,
});

manager.git.commit({
  workspaceName: "ws-api-demo",
  message: "Apply coordinated update",
  addAll: true,
  push: false,
});
```

## Error and Batch Semantics

Validation errors are thrown as JS `TypeError` before workflow execution for
required fields, for example:

- `createWorkspace` requires `name` and `repos`.
- `workspaces.add` requires `workspaceName` and `repoName`.
- `workspaces.merge` requires `workspaceName`.
- `git.commit` requires `message` or `template`.
- `git.branch.create/switch` require `branchName`.

Execution failures (workspace not found, git failures, etc.) throw regular JS
errors from Go workflow/service errors.

Batch-oriented operations return arrays/rows rather than throwing per
repository row failure. Row objects contain success/error signals, for example:

- branch operations: `results[]` with repository-level `success` and `error`.
- rebase status/action: `rows[]` with repository-level state or action result.

## Type Contract

Completion-level declarations are maintained in:

- `pkg/wsmjs/spec/wsm.d.ts.tmpl`
- `pkg/wsmjs/spec/wsm.d.ts` (generated snapshot)

Regenerate and validate with:

```bash
go generate ./pkg/wsmjs/spec
go test ./pkg/wsmjs/spec
```

## Example Script

```javascript
const wsm = require("wsm");
const manager = wsm.createManager({ defaultJobs: 4 });

const repos = manager.registry.listRepositories({});
const workspaces = manager.registry.listWorkspaces();

({
  ok: true,
  version: wsm.version,
  repositories: repos.length,
  workspaces: workspaces.length,
  remote: wsm.consts.remote.ORIGIN,
});
```

## Troubleshooting

| Problem | Cause | Fix |
|---|---|---|
| `Cannot find module 'wsm'` | Script not run via runner | Use `wsm runner <script.js>` |
| Missing result in data mode | Script ends with statement | End with expression object, for example `({ ok: true })` |
| Validation `TypeError` | Required fields missing | Supply required input keys |
| Unexpected mixed human output | Wrong mode | Use `--output-mode data --output json --print-result=false` |

## See Also

- `wsm help wsm-command-reference`
- `wsm help wsm-architecture-overview`
- `wsm help wsm-troubleshooting`
