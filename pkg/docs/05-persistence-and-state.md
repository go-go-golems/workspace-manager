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

Understanding where WSM stores data is essential for debugging issues and for
manual recovery when things go wrong. This page documents every persisted file,
what creates and updates it, and how to safely repair state.

## Where state lives

WSM stores all persistent configuration under your user config directory:

```
~/.config/workspace-manager/
├── registry.json                    # discovered repository catalog
└── workspaces/
    ├── my-feature.json              # workspace definition
    ├── another-workspace.json
    └── ...
```

Workspace directories (the actual worktrees and files you work in) live
separately, typically under a date-stamped root:

```
~/workspaces/
└── 2026-02-28/
    └── my-feature/
        ├── workspace-manager/       # git worktree
        ├── geppetto/                # git worktree
        ├── go.work                  # if Go workspace detected
        └── AGENT.md                 # if agent-source was specified
```

## The registry (`registry.json`)

The registry is a catalog of all git repositories WSM knows about. It maps
repository names to their paths, remote URLs, current branches, and tags.

**Created by**: `wsm discover`
**Updated by**: `wsm discover` (overwrites with fresh scan results)
**Read by**: `wsm create`, `wsm list repos`, `wsm fork`, JS API

When you run `wsm create --repos foo,bar`, WSM looks up `foo` and `bar` in
this registry to find their on-disk paths and remote URLs. If a repository is
not in the registry, the create command fails with a "repositories not found"
error.

To refresh the registry, re-run discovery:

```bash
wsm discover ~/code
```

## Workspace definitions (`workspaces/*.json`)

Each workspace gets its own JSON file containing:

- Workspace name and path
- Branch name and base branch
- List of repositories (name, path, remote URL)
- Creation timestamp
- Whether a Go workspace (`go.work`) was created
- AGENT.md source path (if any)

**Created by**: `wsm create`, `wsm fork`
**Updated by**: `wsm add`, `wsm remove`
**Deleted by**: `wsm delete`
**Read by**: `wsm status`, `wsm info`, `wsm commit`, all git commands

## Workspace directories on disk

The workspace directory contains one subdirectory per repository. Each is a git
worktree (not a full clone), which means it shares the object store with the
original repository. This is fast and space-efficient.

When a workspace includes Go modules that reference each other, WSM
automatically creates a `go.work` file at the workspace root so that local
development works without module replace directives.

If `--agent-source` is specified during create or fork, WSM copies the given
file as `AGENT.md` into the workspace root.

## Write lifecycle

Here is exactly which commands write to which files:

| Command | Writes to |
|---------|-----------|
| `wsm discover` | `registry.json` |
| `wsm create` | New `workspaces/<name>.json` + creates worktree directories |
| `wsm fork` | New `workspaces/<name>.json` + creates worktree directories |
| `wsm add` | Updates existing `workspaces/<name>.json` |
| `wsm remove` | Updates existing `workspaces/<name>.json` |
| `wsm delete` | Removes `workspaces/<name>.json` + optionally deletes directory |
| `wsm merge` | May remove `workspaces/<name>.json` if `--keep-workspace` is false |

No other commands write to disk. Status, diff, log, branch list, and the JS API
are all read-only operations.

## Runtime state vs persisted state

It is important to distinguish what is persisted from what is computed at
runtime:

**Persisted** (survives restarts):
- Repository catalog (`registry.json`)
- Workspace definitions (`workspaces/*.json`)
- The actual files and worktrees on disk

**Computed at runtime** (ephemeral):
- Git status for each repository (dirty/clean, ahead/behind counts)
- Rebase state detection (checks `.git/rebase-merge` and `.git/rebase-apply`)
- Workspace detection from current working directory
- Conflict file lists

If runtime output and persisted files ever seem inconsistent, the persisted
files are the source of truth for workspace membership and configuration, while
the filesystem is the source of truth for actual git state.

## Workspace detection

When you run a command like `wsm status` without specifying a workspace name,
WSM tries to detect which workspace you are in:

1. It loads all workspace JSON files from `~/.config/workspace-manager/workspaces/`.
2. It checks whether your current working directory is inside any known workspace path.
3. If a match is found, that workspace is used.

If detection fails, you get a `not in a workspace directory` error. Pass the
workspace name explicitly to avoid this:

```bash
wsm status my-feature
# or
wsm status --workspace my-feature
```

## Safe maintenance

WSM commands are the safest way to modify state. However, if you need manual
repair:

1. **Back up first**: Copy the JSON file before editing.
2. **Edit conservatively**: Only change the field you need to fix.
3. **Validate after**: Run `wsm list workspaces` and `wsm info <name>` to
   confirm the repair looks correct.

Common manual fixes:
- **Stale workspace pointing to deleted directory**: Delete the workspace JSON
  file from `~/.config/workspace-manager/workspaces/`, or run `wsm delete <name>`.
- **Registry missing a repo**: Re-run `wsm discover` on the parent directory.
- **Workspace JSON has wrong path**: Edit the `path` field to match the actual
  directory location.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Workspace appears in list but path is missing | Workspace JSON exists, directory was deleted manually | Run `wsm delete <name>` to clean up, then recreate |
| Create fails with repo lookup errors | Registry is stale or does not include the repo | Re-run `wsm discover` on the correct directories |
| Status detects wrong workspace | Working directory is inside multiple workspace paths | Pass explicit `--workspace <name>` |
| `go.work` references wrong modules | Repos were added/removed after workspace creation | Delete `go.work` and re-run `wsm add`/`wsm remove` |

## See Also

- `wsm help wsm-getting-started`
- `wsm help wsm-troubleshooting`
- `wsm help wsm-architecture-overview`
