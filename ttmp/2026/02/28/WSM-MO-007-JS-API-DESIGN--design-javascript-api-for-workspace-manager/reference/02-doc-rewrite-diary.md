---
title: "Documentation Rewrite Diary"
date: 2026-02-28
type: diary
topics: [documentation, help-pages, glazed, verification]
---

# Documentation Rewrite Diary

## Step 7: Rewrite Embedded Help Pages (2026-02-28)

### Prompt context

The six embedded help pages in `pkg/docs/` were terse and mechanical. They
read like auto-generated stubs: each had a one-sentence intro, bullet-point
lists of commands, and a small troubleshooting table. The user asked for
engaging, fleshed-out documentation that a human would enjoy reading, with
all information verified against the actual codebase.

### What was done

1. **Full codebase audit.** Read every Cobra command definition in
   `cmd/wsm/cmds/` (registry, workspace, git, js subpackages) to extract
   the exact flag names, types, defaults, and help strings. Cross-referenced
   with the settings structs to confirm nothing was missed.

2. **Verification of existing docs.** Compared every claim in the six existing
   `.md` files against the code. Found multiple omissions:
   - `01-getting-started.md` was missing `--branch-prefix`, `--base-branch`,
     `--agent-source`, `--interactive`, `--dry-run` for create; `--short`,
     `--untracked`, `--jobs` for status; `--force`, `--force-worktrees` for delete.
   - `02-command-reference.md` listed commands as bare bullets without flags.
     Missing: commit flags (`-m`, `--interactive`, `--add-all`, `--push`,
     `--dry-run`, `--commit-template`), diff flags (`--staged`, `--repo`,
     `--jobs`), log flags (`--since`, `--oneline`, `--limit`), branch flags
     (`--track`), rebase flags (`--target`, `--interactive`, `--jobs`,
     `--manual`), delete flags (`--force`, `--force-worktrees`,
     `--remove-files`), merge flags (`--dry-run`, `--force`,
     `--keep-workspace`), fork flags (`--branch`, `--branch-prefix`,
     `--agent-source`, `--dry-run`), info flag (`--field`).
   - `03-js-api-and-runner.md` was missing manager-level methods:
     `discover`, `listWorkspaces`, `listRepositories`. The grouped
     namespaces were mentioned but not detailed. Input types were absent.
     Constants were incomplete (missing `remoteRefKind` sub-object).
   - `04-architecture-overview.md` mentioned layer names but had no package
     paths, no workflow list, no explanation of the git client hybrid model,
     and no contributor guidance.
   - `05-persistence-and-state.md` had no concrete file examples, no
     explanation of the date-based workspace root, and no write lifecycle table.
   - `06-troubleshooting.md` had a flat table instead of categorized sections,
     no rebase recovery walkthrough, and no runner troubleshooting.

3. **Rewrote all six pages.** Each page now:
   - Opens with context about why the page matters
   - Documents every flag with type, default, and description
   - Includes working examples
   - Has structured troubleshooting sections organized by failure category
   - Cross-references related pages

4. **Created verification script.** `scripts/verify-doc-flags.sh` runs
   `--help` on every command and subcommand to compare against documentation.

### What worked

- Reading the Cobra command definitions directly was the single most reliable
  source of truth. Every flag is defined with `fields.New(...)` and the
  settings struct tags confirm the mapping.
- The existing doc infrastructure (Glazed help system with YAML frontmatter,
  `go:embed`, slug-based lookup) worked seamlessly -- no changes to `doc.go`
  or `doc_test.go` were needed.

### What was tricky

- Some commands accept workspace name both as a positional argument AND as
  `--workspace` flag. This dual-path had to be documented without confusion.
- The `wsm list` parent command has subcommands (`repos`, `workspaces`) that
  live in separate files (`list_repos.go`, `list_workspaces.go`) wired through
  `root.go`. The parent grouping is done by Cobra, not Glazed.
- The JS API has both flat methods and grouped namespaces that call the same
  implementation. Documenting both without it looking duplicated required a
  clear table layout.

### What needs a second pair of eyes

- The workspace root path convention (`~/workspaces/YYYY-MM-DD/`) -- I derived
  this from reading `workspace.go` but did not find a user-configurable override.
  Confirm this is accurate.
- The `go.work` auto-detection behavior -- it seems to be created when Go
  modules are detected among the workspace repos, but the exact trigger
  conditions should be verified.
- The `--agent-source` flag inheritance during fork -- I documented that fork
  copies the source workspace's AGENT.md by default, based on reading fork.go.

### Files changed

- `pkg/docs/01-getting-started.md` -- full rewrite
- `pkg/docs/02-command-reference.md` -- full rewrite
- `pkg/docs/03-js-api-and-runner.md` -- full rewrite
- `pkg/docs/04-architecture-overview.md` -- full rewrite
- `pkg/docs/05-persistence-and-state.md` -- full rewrite
- `pkg/docs/06-troubleshooting.md` -- full rewrite
- `scripts/verify-doc-flags.sh` -- new verification script

### Code review instructions

1. Run `go test ./pkg/docs/` to confirm all six slugs still load.
2. Build `wsm` and run `wsm help wsm-getting-started` (and each other slug)
   to confirm rendering.
3. Spot-check flags in the command reference against `wsm <command> --help`.
4. Read through the JS API page and compare against `pkg/wsmjs/module/module.go`.
