---
Title: Glazed CLI Migration Implementation Plan v2
Ticket: WSM-MO-006-GLAZED-CLI-MIGRATION
Status: active
Topics:
    - architecture
    - refactor
    - workspace-manager
    - glazed
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/wsm/root.go
      Note: Current root and registration to replace
    - Path: cmd/cmds
      Note: Legacy command location to be removed
    - Path: pkg/wsm/workflows
      Note: Reusable orchestration services to keep as command backend
ExternalSources:
    - /home/manuel/.codex/skills/glazed-command-authoring/SKILL.md
Summary: v2 migration plan with hard directory layout, human-first dual output policy, and minimal section strategy for Glazed command porting.
LastUpdated: 2026-02-28T16:42:00-05:00
WhatFor: Define the concrete v2 direction for Glazed migration with simplified section architecture and final command layout.
WhenToUse: Use this v2 doc for implementation; it supersedes section/layout decisions in v1.
---

# Glazed CLI Migration Implementation Plan v2

## Executive Summary

This v2 plan locks in three decisions:

1. **Directory layout**: all commands move to `cmd/wsm/cmds/<group>/<verb>.go`; legacy `cmd/cmds` is removed.
2. **Output policy**: every command is **dual output** capable, with **human output as default**.
3. **Section strategy**: keep sections intentionally few (not tiny/micro sections). Some flag duplication between commands is acceptable.

The migration still keeps business logic in `pkg/wsm` and `pkg/wsm/workflows`.

## Problem Statement

The v1 migration plan was directionally correct but left room for drift in:

- command file layout,
- how dual output should work by default,
- how far to decompose sections.

Without pinning these, implementation can splinter into over-engineered section abstractions and inconsistent command placement.

## Proposed Solution

## 1) Hard Layout Decision

Target layout:

- `cmd/wsm/root.go` (root wiring only)
- `cmd/wsm/cmds/<group>/root.go` (group registration)
- `cmd/wsm/cmds/<group>/<verb>.go` (single verb per file)

Legacy location to delete after parity:

- `cmd/cmds/*`

Proposed groups:

- `cmd/wsm/cmds/registry`
  - `discover.go`
  - `list_repos.go`
  - `list_workspaces.go`
- `cmd/wsm/cmds/workspace`
  - `create.go`, `fork.go`, `merge.go`, `add.go`, `remove.go`, `delete.go`, `info.go`, `status.go`
- `cmd/wsm/cmds/git`
  - `commit.go`, `branch_create.go`, `branch_switch.go`, `branch_list.go`, `rebase.go`, `rebase_status.go`, `rebase_continue.go`, `rebase_abort.go`, `diff.go`, `log.go`

## 2) Dual Output, Human First

All commands implement one shared output contract:

- `human` channel: narrative/status lines for interactive users.
- `data` channel: structured rows for machine use.

Default behavior:

- `output_mode = human` (human-first).

Supported modes:

- `human`: print human channel only.
- `data`: emit rows only.
- `both`: human first, then rows.

Suggested common flag:

- `--output-mode human|data|both` (default `human`).

Interaction with Glazed output settings:

- In `data`/`both`, Glazed output format (`--output json|yaml|table...`) applies to row stream.
- In `human`, Glazed row output can be skipped.

## 3) Minimal Section Strategy (No Tiny Sections)

We intentionally avoid many tiny sections.

### Option A (Recommended): Two reusable sections

1. **runtime section**
- `output-mode`, `jobs`, `dry-run`, `interactive`, `force` (fields included per command as needed).

2. **scope section**
- `workspace`, `repo`, `branch`, `target`, `since`, `limit` (again, included per command as needed).

Plus standard Glazed sections:

- `settings.NewGlazedSchema()`
- `cli.NewCommandSettingsSection()`

### Option B (Even simpler): One reusable section

1. **runtime section only**

Everything else is defined directly in each command descriptor. This permits some duplicate flags and keeps each command self-contained.

### Decision

Start with **Option B** for speed and clarity. If duplication becomes painful, introduce Option A in a later pass. Duplicate flags are acceptable right now.

## Design Decisions

1. Keep command-specific flag definitions local even when duplicated, unless duplication clearly harms maintenance.
2. Keep `pkg/wsm` workflows as execution source; Glazed commands are adapters.
3. Human-first defaults are intentional product behavior, not temporary migration scaffolding.
4. Grouped subcommands (`list`, `branch`, `rebase`) become separate verb files under their group.

## Alternatives Considered

1. Deep section decomposition (many tiny sections).
- Rejected: high abstraction overhead, hard to reason about, little immediate benefit.

2. Data-first output default.
- Rejected: current users are primarily interactive; human-first gives better UX continuity.

3. Keep old `cmd/cmds` and add new path in parallel.
- Rejected: prolongs migration and creates source-of-truth ambiguity.

## Command-by-Command Port Plan (v2)

Each command below maps to `cmd/wsm/cmds/<group>/<verb>.go`.

### registry/discover

- Delegate: `workflows.NewDiscoverWorkflow().Discover(...)`
- Human: discovered paths + total repositories.
- Data rows: `{path, repository_count}` summary row.
- Flags: `paths`, `recursive`, `max-depth`, `output-mode`.

### registry/list_repos

- Delegate: `workflows.NewListWorkflow().ListRepositories(tags)`
- Human: table-ish summary line + repo count.
- Data rows: one row per repo.
- Flags: `tags`, `output-mode` (drop custom `--format` eventually).

### registry/list_workspaces

- Delegate: `workflows.NewListWorkflow().ListWorkspaces()`
- Human: workspace count + concise list.
- Data rows: one row per workspace.
- Flags: `output-mode`.

### workspace/create

- Delegate: `workflows.NewCreateWorkflow().Create(...)`
- Human: existing create summary/next-steps.
- Data rows: workspace result row.
- Flags: `repos`, `branch`, `branch-prefix`, `base-branch`, `agent-source`, `interactive`, `dry-run`, `output-mode`.

### workspace/fork

- Delegate: `workflows.NewForkWorkflow().Plan/Fork(...)`
- Human: source/base branch plan + result summary.
- Data rows: fork plan/result rows.
- Flags: `workspace`, `branch`, `branch-prefix`, `agent-source`, `dry-run`, `output-mode`.

### workspace/merge

- Delegate: `workflows.NewMergeWorkflow().Execute(...)`
- Human: merge progress and safety confirmations.
- Data rows: final summary row (+ optional per-repo event rows later).
- Flags: `workspace`, `dry-run`, `force`, `keep-workspace`, `output-mode`.

### workspace/add

- Delegate: `WorkspaceManager.AddRepositoryToWorkspace(...)`
- Human: action summary.
- Data rows: result row.
- Flags: `branch`, `force`, `output-mode`.

### workspace/remove

- Delegate: `WorkspaceManager.RemoveRepositoryFromWorkspace(...)`
- Human: action summary.
- Data rows: result row.
- Flags: `force`, `remove-files`, `output-mode`.

### workspace/delete

- Delegate: `workflows.NewDeleteWorkflow().Preview/Delete(...)`
- Human: preview + confirm + outcome.
- Data rows: preview row and completion row.
- Flags: `force`, `force-worktrees`, `remove-files`, `output-mode`.

### workspace/info

- Delegate: `workflows.NewInfoWorkflow().ResolveWorkspace/FieldValue(...)`
- Human: compact info block.
- Data rows: workspace row or single-field row.
- Flags: `workspace`, `field`, `output-mode`.

### workspace/status

- Delegate: `workflows.NewStatusWorkflow().GetStatus(...)`
- Human: current detailed/short rendering.
- Data rows: one row per repository status.
- Flags: `workspace`, `short`, `untracked`, `jobs`, `output-mode`.

### git/commit

- Delegate: `workflows.NewCommitWorkflow().Prepare/Execute(...)`
- Human: commit selection/progress summary.
- Data rows: completion row (`repo_count`, `push`, `dry_run`).
- Flags: `message`, `template`, `interactive`, `add-all`, `push`, `dry-run`, `output-mode`.

### git/branch_create

- Delegate: `wsm.NewBranchOperations(...).CreateBranch(...)`
- Human: per-repo branch status lines.
- Data rows: per-repo operation rows.
- Flags: `track`, `output-mode`.

### git/branch_switch

- Delegate: `wsm.NewBranchOperations(...).SwitchBranch(...)`
- Human: per-repo switch status lines.
- Data rows: per-repo operation rows.
- Flags: `output-mode`.

### git/branch_list

- Delegate: existing branch listing op path.
- Human: grouped branch list by repo.
- Data rows: per-repo/branch rows.
- Flags: `output-mode`.

### git/rebase

- Delegate: `workflows.NewRebaseWorkflow().Rebase(...)`
- Human: rebase plan/results and manual instructions.
- Data rows: `RebaseResult` rows.
- Flags: `target`, `interactive`, `dry-run`, `jobs`, `manual`, `output-mode`.

### git/rebase_status

- Delegate: `workflow.Status(...)`
- Human: status table.
- Data rows: `RebaseStatusRow` rows.
- Flags: `repo`, `jobs`, `output-mode`.

### git/rebase_continue

- Delegate: `workflow.Continue(...)`
- Human: action summary.
- Data rows: `RebaseActionRow` rows.
- Flags: `repo`, `jobs`, `output-mode`.

### git/rebase_abort

- Delegate: `workflow.Abort(...)`
- Human: action summary.
- Data rows: `RebaseActionRow` rows.
- Flags: `repo`, `jobs`, `output-mode`.

### git/diff

- Delegate: `GitOperations.GetDiff(...)`
- Human: diff text by repo.
- Data rows: row with `{repo, diff}` or chunk rows.
- Flags: `staged`, `repo`, `jobs`, `output-mode`.

### git/log

- Delegate: `HistoryOperations.GetWorkspaceLog(...)`
- Human: commit log rendering.
- Data rows: commit rows.
- Flags: `since`, `oneline`, `limit`, `output-mode`.

## Implementation Plan

### Phase A: Layout Cutover Scaffold

1. Create `cmd/wsm/cmds/<group>` directories and group roots.
2. Add Glazed command constructors per group/verb file.
3. Wire root registration to new group roots.

### Phase B: Low-Risk Commands First

- `discover`, `list_*`, `info`, `add`, `remove`.

### Phase C: Workspace/Git Medium Commands

- `status`, `diff`, `log`, `branch_*`, `create`, `fork`, `commit`, `delete`.

### Phase D: High-Risk Commands

- `rebase*`, `merge`.

### Phase E: Removal and Cleanup

1. Delete `cmd/cmds`.
2. Remove deprecated root init paths.
3. Keep only `cmd/wsm/cmds/<group>/<verb>.go` layout.
4. Update docs/help and verify parity.

## Open Questions

1. Should `output-mode=both` be default for CI-focused subcommands, or keep `human` globally?
2. For text-heavy diff/log output, should we define a canonical row schema now or in a follow-up ticket?

## References

- `/home/manuel/.codex/skills/glazed-command-authoring/SKILL.md`
- `/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/root.go`
- `/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workflows`
