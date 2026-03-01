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

WSM (workspace-manager) orchestrates multi-repository development. If you work on
features that span two or more Git repositories at once, WSM keeps them aligned
under a single workspace: same branch name, shared status view, coordinated
commits and rebases.

This tutorial walks you through the core loop: discover your repositories,
create a workspace, work in it, and clean up when you are done.

## Prerequisites

WSM should be installed and on your `$PATH`. Confirm with:

```bash
wsm --help
```

Your repositories must already be cloned somewhere on disk. WSM does not clone
for you -- it discovers existing repositories and creates lightweight git
worktrees to avoid duplicating them.

## Step 1: Discover repositories

Before WSM can manage your repos, it needs to know where they are. Point
`wsm discover` at one or more parent directories:

```bash
wsm discover ~/code ~/projects
```

Discovery walks each path recursively (up to depth 3 by default) looking for
`.git` directories. Found repositories are recorded in a local registry at
`~/.config/workspace-manager/registry.json`.

You can tune the scan:

| Flag | Default | Purpose |
|------|---------|---------|
| `-r, --recursive` | `true` | Recurse into subdirectories |
| `--max-depth` | `3` | Maximum recursion depth |

After discovery, verify your registry:

```bash
wsm list repos
```

This prints a table of repository name, path, current branch, tags, and remote
URL. If a repository is missing, re-run discovery with a wider path or higher
`--max-depth`.

## Step 2: Create a workspace

A workspace ties together a set of repositories under a common branch. To create
one, name it and list the repos:

```bash
wsm create my-feature --repos workspace-manager,geppetto
```

WSM will:
1. Create a date-stamped workspace directory (e.g. `~/workspaces/2026-02-28/my-feature/`).
2. Set up a git worktree for each named repository inside that directory.
3. Create or check out a branch in each worktree.

If you do not supply `--branch`, WSM generates one automatically as
`<prefix>/<workspace-name>`. The default prefix is `task`, so the branch above
would be `task/my-feature`. You can change this:

| Flag | Default | Purpose |
|------|---------|---------|
| `--branch` | *(auto)* | Explicit branch name for all repos |
| `--branch-prefix` | `task` | Prefix when auto-generating branches |
| `--base-branch` | *(repo default)* | Branch to create from |
| `--repos` | *(required)* | Comma-separated repository names |
| `--interactive` | `false` | Pick repositories interactively |
| `--dry-run` | `false` | Preview what would be created |
| `--agent-source` | | Copy an AGENT.md template into the workspace |

If you are not sure which repos to include, try `--interactive` to get a
selection prompt, or `--dry-run` to preview the result before committing to it.

After creation, `cd` into the workspace and start working:

```bash
cd ~/workspaces/2026-02-28/my-feature
```

## Step 3: Check status

`wsm status` gives you a consolidated view of git state across every repository
in the workspace. Run it from inside the workspace:

```bash
wsm status
```

Or pass the workspace name explicitly from anywhere:

```bash
wsm status my-feature
```

The output shows a per-repository table with columns for branch, status,
changes, sync state, merge state, and rebase state. Below the table, modified
and staged files are listed per repository.

Useful flags:

| Flag | Default | Purpose |
|------|---------|---------|
| `--short` | `false` | One-line-per-repo summary |
| `--untracked` | `false` | Include untracked files in output |
| `--jobs` | `1` | Process repositories in parallel |

Use `--short` for a quick glance and `--untracked` when you want to see new
files that have not been added yet.

## Step 4: Clean up when done

Once your feature branches are merged upstream, delete the workspace:

```bash
wsm delete my-feature
```

WSM will show you the current workspace status, then ask for confirmation before
proceeding. It removes git worktrees and the workspace configuration file.

Key flags:

| Flag | Default | Purpose |
|------|---------|---------|
| `--remove-files` | `false` | Also delete the workspace directory and all contents |
| `--force-worktrees` | `false` | Force-remove worktrees even with uncommitted changes |
| `-f, --force` | `false` | Skip the confirmation prompt |

Without `--remove-files`, WSM removes worktree registrations and config but
leaves the directory on disk. Add `--remove-files` for a full cleanup. Use
`--force-worktrees` only when you are certain that any uncommitted work can be
discarded.

## What to do next

Now that you have the basic workflow, explore these areas:

- **Commit across repos**: `wsm commit -m "your message"` stages and commits
  changes across all workspace repositories at once. See `wsm help wsm-command-reference`.

- **Rebase all repos**: `wsm rebase` rebases every repository against a target
  branch (default `main`). If conflicts arise in one repo, the others still
  proceed, and you can resolve conflicts per-repo with `wsm rebase status` and
  `wsm rebase continue`.

- **Fork a workspace**: `wsm fork new-name` creates a new workspace from an
  existing one, preserving the same repository set but creating fresh branches.

- **Script with JavaScript**: `wsm runner script.js` executes JavaScript with
  the full WSM API available via `require("wsm")`. See `wsm help wsm-js-api-and-runner`.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `repositories not found` during create | Discovery did not include target paths | Re-run `wsm discover` on the correct directories |
| `not in a workspace directory` on status | Running outside workspace without explicit name | Pass the workspace name as an argument or use `--workspace <name>` |
| Create seems to produce stale results | Old workspace JSON or manual filesystem edits | Check `~/.config/workspace-manager/workspaces/` and delete stale JSON files |

## See Also

- `wsm help wsm-command-reference`
- `wsm help wsm-persistence-and-state`
- `wsm help wsm-troubleshooting`
