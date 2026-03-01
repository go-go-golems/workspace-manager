---
Title: Getting Started with WSM
Slug: wsm-getting-started
Short: Fast path to discover repositories, create a workspace, and check status.
Topics:
- workspace-manager
- getting-started
- workflow
Commands:
- discover
- create
- status
- delete
Flags:
- --repos
- --branch
- --dry-run
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This page covers the shortest reliable path to become productive with WSM. It shows the minimal command sequence to discover repositories, create a workspace, and inspect status.

This flow matters because WSM is most useful when the repository registry is accurate and workspace creation is intentional.

## Step 1: Discover repositories

Run discovery on your development folders.

```bash
wsm discover ~/code ~/projects --recursive --max-depth 3
```

If discovery succeeds, WSM stores repository metadata in your config directory and later commands can reference repos by name.

## Step 2: Create a workspace

Create a workspace with explicit repositories.

```bash
wsm create my-feature --repos workspace-manager,geppetto --branch feature/my-feature
```

If you prefer branch auto-generation, omit `--branch` and use `--branch-prefix`.

## Step 3: Check status

Run status from inside the workspace or pass the workspace name.

```bash
wsm status
# or
wsm status --workspace my-feature
```

Use this as your primary sanity check before commit/rebase workflows.

## Step 4: Clean up when done

Delete the workspace once work is merged.

```bash
wsm delete my-feature --remove-files
```

Use `--force-worktrees` only when you know local worktree state is safe to discard.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `repositories not found` during create | Discovery did not include target paths | Re-run `wsm discover` on correct directories |
| `not in a workspace directory` on status | Running outside workspace and no workspace flag | Pass `--workspace <name>` |
| Create/delete operations seem inconsistent | Stale workspace JSON or manual filesystem changes | Verify persisted files under `~/.config/workspace-manager/workspaces/` |

## See Also

- `wsm help wsm-command-reference`
- `wsm help wsm-persistence-and-state`
- `wsm help wsm-troubleshooting`
