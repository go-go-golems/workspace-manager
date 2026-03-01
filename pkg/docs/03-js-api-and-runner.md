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

This page covers how to automate workspace-manager via JavaScript. It explains the runner command and the current JS API surface.

This matters when you want reusable scripts without shelling out to many CLI commands.

## Run a script

```bash
wsm runner demo/js/wsm-api-smoke.js
```

For structured output:

```bash
wsm runner demo/js/wsm-api-smoke.js --output-mode data --print-result=false
```

## API surface

The runner pre-registers `require("wsm")`.

Current top-level exports:

- `wsm.version`
- `wsm.consts`
- `wsm.createManager(options)`
- `wsm.discover(input)`
- `wsm.createWorkspace(input)`
- `wsm.status(input)`

Manager exports include grouped namespaces:

- `manager.registry.listRepositories(...)`
- `manager.workspaces.create|list|status(...)`
- `manager.git.status(...)`

## Example

```javascript
const wsm = require("wsm");

const manager = wsm.createManager({ defaultJobs: 8 });
const repositories = manager.registry.listRepositories({ tags: [] });

({
  version: wsm.version,
  repositoryCount: repositories.length,
  defaultRemote: wsm.consts.remote.ORIGIN,
});
```

## Design notes

- API methods route through workflow-backed Go facades, not command wrappers.
- Constants are exported from typed branch enums.
- Result conversion currently uses JSON portability conversion for predictable JS values.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `Cannot find module 'wsm'` | Script not run through `wsm runner` | Run via `wsm runner <script.js>` |
| Empty or missing result in human output | Script returns `undefined`/`null` | Return an object literal at script end |
| Unexpected data in automations | Human mode enabled | Use `--output-mode data` |

## See Also

- `wsm help wsm-command-reference`
- `wsm help wsm-architecture-overview`
- `wsm help wsm-troubleshooting`
