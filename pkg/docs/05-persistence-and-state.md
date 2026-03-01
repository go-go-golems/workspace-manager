---
Title: WSM Persistence and State
Slug: wsm-persistence-and-state
Short: Exact locations and lifecycle of registry, workspace, and runtime state.
Topics:
- workspace-manager
- state
- operations
Commands:
- discover
- create
- delete
- status
Flags:
- --workspace
- --remove-files
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page covers where WSM persists state and what each state file means. It is the source of truth for operational debugging.

This matters because many support issues are caused by stale or manually-edited state files.

## Persisted state locations

WSM uses your user config directory:

- Registry: `~/.config/workspace-manager/registry.json`
- Workspace definitions: `~/.config/workspace-manager/workspaces/<workspace>.json`

Workspace directories are created separately on disk (for example under your configured workspace root).

## Write lifecycle

- `wsm discover` updates `registry.json`.
- `wsm create` writes a new workspace JSON file.
- `wsm add`/`wsm remove` update existing workspace JSON.
- `wsm delete` removes workspace JSON and optionally workspace files.

## Runtime state vs persisted state

- Runtime state: command execution, git status snapshots, in-memory workflow decisions.
- Persisted state: registry and workspace JSON files.

If runtime output and persisted files diverge, inspect filesystem paths first.

## Safe maintenance guidance

- Prefer WSM commands to mutate state instead of manual JSON edits.
- If manual repair is necessary, back up JSON files before editing.
- After repair, run `wsm list workspaces` and `wsm status --workspace <name>`.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Workspace appears in list but path is missing | Workspace JSON exists, filesystem folder removed manually | Delete stale workspace with `wsm delete <name>` then recreate |
| Create fails with repo lookup errors | Registry is stale | Re-run `wsm discover` and retry create |
| Status detects wrong workspace | CWD-based detection ambiguity | Pass explicit `--workspace <name>` |

## See Also

- `wsm help wsm-getting-started`
- `wsm help wsm-troubleshooting`
- `wsm help wsm-architecture-overview`
