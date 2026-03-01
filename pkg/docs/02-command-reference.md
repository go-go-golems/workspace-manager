---
Title: WSM Command Reference
Slug: wsm-command-reference
Short: Concise reference for core workspace, registry, git, and JS runner commands.
Topics:
- workspace-manager
- commands
- reference
Commands:
- discover
- list
- create
- status
- commit
- rebase
- runner
Flags:
- --workspace
- --jobs
- --output-mode
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page is a complete reference for every WSM command, organized by group.
Every flag listed here is verified against the source code. If you are looking
for a guided walkthrough, start with `wsm help wsm-getting-started` instead.

All commands accept the global `--output-mode` flag (see Output Modes at the
bottom of this page).

---

## Registry commands

These commands manage the repository registry that WSM uses to resolve repo
names.

### `wsm discover [paths...]`

Scan directories for git repositories and add them to the registry.

```bash
wsm discover ~/code ~/projects
wsm discover ~/monorepo --max-depth 1
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `paths` | argument (list) | cwd | Directories to scan |
| `-r, --recursive` | bool | `true` | Recurse into subdirectories |
| `--max-depth` | int | `3` | Maximum recursion depth |

### `wsm list repos`

List all discovered repositories.

```bash
wsm list repos
wsm list repos --tags go,cli
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--tags` | string list | | Filter by tags (comma-separated) |

### `wsm list workspaces`

List all created workspaces, sorted newest first.

```bash
wsm list workspaces
wsm list workspaces --output-mode data
```

No command-specific flags.

---

## Workspace commands

These commands manage the lifecycle of multi-repository workspaces.

### `wsm create <name>`

Create a new workspace with git worktrees for the selected repositories.

```bash
wsm create my-feature --repos wsm,geppetto
wsm create hotfix --repos wsm --branch hotfix/urgent
wsm create spike --interactive --dry-run
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `workspace-name` | argument | *(required)* | Name for the new workspace |
| `--repos` | string list | | Repository names to include |
| `--branch` | string | *(auto)* | Explicit branch name |
| `--branch-prefix` | string | `task` | Prefix for auto-generated branches |
| `--base-branch` | string | *(repo default)* | Base branch to create from |
| `--agent-source` | string | | Path to AGENT.md template to copy in |
| `--interactive` | bool | `false` | Choose repositories interactively |
| `--dry-run` | bool | `false` | Preview without creating |

When `--branch` is omitted, WSM generates `<branch-prefix>/<workspace-name>`.

### `wsm info [name]`

Display metadata about a workspace.

```bash
wsm info
wsm info my-feature --field path
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `workspace-name` | argument | *(auto-detect)* | Workspace name |
| `--workspace` | string | | Workspace name (alternative) |
| `--field` | string | | Output a single field: `path`, `name`, `branch`, `repositories`, `created`, `date`, `time` |

Use `--field path` in scripts to get the workspace directory without parsing
human output.

### `wsm status [name]`

Show aggregated git status across all workspace repositories.

```bash
wsm status
wsm status my-feature --short
wsm status --untracked --jobs 4
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `workspace-name` | argument | *(auto-detect)* | Workspace name |
| `--workspace` | string | | Workspace name (alternative) |
| `--short` | bool | `false` | One-line-per-repo summary |
| `--untracked` | bool | `false` | Include untracked files |
| `--jobs` | int | `1` | Parallel repository processing |

### `wsm add <workspace> <repo>`

Add a repository to an existing workspace.

### `wsm remove <workspace> <repo>`

Remove a repository from an existing workspace.

### `wsm fork <new-name> [source]`

Create a new workspace by forking an existing one. Inherits the same repository
set, creates fresh branches.

```bash
wsm fork iteration-2
wsm fork v2-spike my-feature --branch spike/v2 --dry-run
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `new-workspace-name` | argument | *(required)* | Name for the new workspace |
| `source-workspace-name` | argument | *(auto-detect)* | Source workspace |
| `--workspace` | string | | Source workspace (alternative) |
| `--branch` | string | *(auto)* | Branch for the new workspace |
| `--branch-prefix` | string | `task` | Prefix for auto-generated branches |
| `--agent-source` | string | | AGENT.md template (defaults to source workspace's) |
| `--dry-run` | bool | `false` | Preview without creating |

### `wsm merge [name]`

Merge workspace branches back into their base branch.

```bash
wsm merge my-feature
wsm merge --dry-run
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `workspace-name` | argument | *(auto-detect)* | Workspace name |
| `--workspace` | string | | Workspace name (alternative) |
| `--dry-run` | bool | `false` | Show what would be merged |
| `--force` | bool | `false` | Skip confirmation prompt |
| `--keep-workspace` | bool | `false` | Keep workspace after merge |

### `wsm delete <name>`

Delete a workspace registration and optionally its files.

```bash
wsm delete old-feature
wsm delete old-feature --remove-files -f
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `workspace-name` | argument | | Workspace name |
| `--workspace` | string | | Workspace name (alternative) |
| `-f, --force` | bool | `false` | Skip confirmation prompt |
| `--force-worktrees` | bool | `false` | Force-remove worktrees even with uncommitted changes |
| `--remove-files` | bool | `false` | Delete the workspace directory and all contents |

---

## Git commands

These commands operate on the repositories in the current (or specified)
workspace, applying git operations across all of them.

### `wsm commit`

Commit changes across workspace repositories with a single message.

```bash
wsm commit -m "feat: add shared validation"
wsm commit --add-all --push -m "fix: correct timestamps"
wsm commit --interactive
wsm commit --dry-run -m "wip"
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-m, --message` | string | | Commit message |
| `--interactive` | bool | `false` | Interactively select files to commit |
| `--add-all` | bool | `false` | Stage all changes before committing |
| `--push` | bool | `false` | Push to remote after committing |
| `--dry-run` | bool | `false` | Show what would be committed |
| `--commit-template` | string | | Use a commit message template |

If `--message` is not provided and `--interactive` is not set, the command
errors. In interactive mode, WSM shows changes per-repo and lets you confirm
before proceeding.

### `wsm diff`

Show a unified diff across all workspace repositories.

```bash
wsm diff
wsm diff --staged
wsm diff --repo geppetto
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--staged` | bool | `false` | Show staged changes only |
| `--repo` | string | | Filter to a specific repository |
| `--jobs` | int | `1` | Parallel processing |

### `wsm log`

Show commit history across all workspace repositories.

```bash
wsm log
wsm log --since "1 week ago" --oneline
wsm log --limit 5
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--since` | string | | Show commits since date (e.g. `"1 week ago"`) |
| `--oneline` | bool | `false` | One line per commit |
| `--limit` | int | `10` | Max commits per repository |

### `wsm branch create <name>`

Create a branch across all workspace repositories.

```bash
wsm branch create feature/shared-types
wsm branch create feature/shared-types --track
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `branch-name` | argument | *(required)* | Branch name to create |
| `--track` | bool | `false` | Set up remote tracking |

### `wsm branch switch <name>`

Switch all workspace repositories to the given branch.

```bash
wsm branch switch main
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `branch-name` | argument | *(required)* | Branch to switch to |

### `wsm branch list`

Show the current branch for each repository in the workspace.

```bash
wsm branch list
```

No command-specific flags.

### `wsm rebase [repository]`

Rebase workspace repositories onto a target branch. By default, rebases all
repositories against `main`.

```bash
wsm rebase
wsm rebase --target develop
wsm rebase geppetto
wsm rebase --manual
wsm rebase --jobs 4 --dry-run
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `repository` | argument | *(all)* | Rebase only this repository |
| `--target` | string | `main` | Target branch to rebase onto |
| `--dry-run` | bool | `false` | Show what would happen |
| `-i, --interactive` | bool | `false` | Interactive rebase |
| `--jobs` | int | `1` | Parallel processing |
| `--manual` | bool | `false` | Print git commands instead of running them |

Manual mode is useful when you want to review the exact git commands before
executing them yourself.

### `wsm rebase status`

Check rebase state and conflict counts across repositories.

```bash
wsm rebase status
wsm rebase status --repo geppetto
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--repo` | string | | Check only this repository |
| `--jobs` | int | `1` | Parallel processing |

### `wsm rebase continue`

Continue in-progress rebases after resolving conflicts.

```bash
wsm rebase continue
wsm rebase continue --repo geppetto
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--repo` | string | | Continue only this repository |
| `--jobs` | int | `1` | Parallel processing |

### `wsm rebase abort`

Abort in-progress rebases across repositories.

```bash
wsm rebase abort
wsm rebase abort --repo geppetto
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--repo` | string | | Abort only this repository |
| `--jobs` | int | `1` | Parallel processing |

---

## JavaScript commands

### `wsm runner <script.js>`

Execute a JavaScript file with the WSM API pre-loaded. The script can
`require("wsm")` to access workspace management functions.

```bash
wsm runner demo/js/wsm-api-smoke.js
wsm runner my-automation.js --print-result=false --output-mode data
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `script` | argument | *(required)* | Path to JavaScript file |
| `--print-result` | bool | `true` | Print the script's return value |

The script's final expression is its return value. When `--print-result` is
true (the default), this value is printed as formatted output. Use
`--output-mode data` for machine-readable output.

High-level API groups available from `require("wsm")`:

- `manager.registry` (`listRepositories`, `listWorkspaces`)
- `manager.workspaces` (`create`, `list`, `status`, `info`, `add`, `remove`, `delete`, `fork`, `merge`)
- `manager.git` (`status`, `commit`, `diff`, `log`, `branch.*`, `rebase.*`)
- `manager.loadWorkspace(name)` for workspace-handle scoped methods

See `wsm help wsm-js-api-and-runner` for full method contracts and input shapes.

---

## Output modes

Every command supports the `--output-mode` flag:

| Mode | Description |
|------|-------------|
| `human` | Human-readable formatted output (default) |
| `data` | Structured table output via Glazed (machine-friendly) |
| `both` | Human output first, then structured data |

Use `data` or `both` when piping WSM output into scripts or other tools. Glazed
supports additional output formatting flags (JSON, YAML, CSV, etc.) when
`--output-mode` is `data` or `both`.

## See Also

- `wsm help wsm-getting-started`
- `wsm help wsm-js-api-and-runner`
- `wsm help wsm-architecture-overview`
