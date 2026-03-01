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

This page covers the command surface most users need daily. It summarizes what each command group does and where to go deeper.

This matters because WSM has both workspace lifecycle commands and multi-repo git orchestration commands.

## Registry commands

- `wsm discover [paths...]` discovers git repositories.
- `wsm list repos` lists discovered repositories.
- `wsm list workspaces` lists persisted workspaces.

## Workspace commands

- `wsm create <name> --repos <r1,r2,...>` creates workspace worktrees.
- `wsm info [name]` shows workspace metadata.
- `wsm status [name]` shows aggregated git status.
- `wsm add <workspace> <repo>` and `wsm remove <workspace> <repo>` mutate membership.
- `wsm fork <new-name> [source]` forks workspace topology.
- `wsm merge [workspace]` merges branch work back.
- `wsm delete <workspace>` removes workspace registration and optionally files.

## Git commands

- `wsm commit` coordinates commits across repositories.
- `wsm diff` and `wsm log` aggregate read-only git output.
- `wsm branch create|switch|list` runs branch-level operations.
- `wsm rebase` and `wsm rebase status|continue|abort` manage multi-repo rebase flows.

## JS commands

- `wsm runner <script.js>` executes JavaScript with `require("wsm")` pre-registered.

## Output modes

Many commands support `--output-mode`:

- `human`: human-readable output.
- `data`: structured table output.
- `both`: human first, then data.

Use `data` or `both` for automation pipelines.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Command not found in docs | Command may be grouped under another section | Run `wsm --help` then inspect subgroup |
| Inconsistent output in scripts | Mixed human/data output mode | Force `--output-mode data` |
| Rebase commands fail for a repo | Repo-specific conflict or missing branch | Run `wsm rebase status --repo <name>` first |

## See Also

- `wsm help wsm-getting-started`
- `wsm help wsm-js-api-and-runner`
- `wsm help wsm-architecture-overview`
