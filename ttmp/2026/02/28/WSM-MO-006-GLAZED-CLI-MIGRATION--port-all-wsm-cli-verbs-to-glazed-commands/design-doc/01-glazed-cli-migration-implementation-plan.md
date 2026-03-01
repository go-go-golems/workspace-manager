---
Title: Glazed CLI Migration Implementation Plan
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
      Note: Current Cobra root wiring and command registration
    - Path: cmd/cmds/cmd_add.go
    - Path: cmd/cmds/cmd_branch.go
    - Path: cmd/cmds/cmd_commit.go
    - Path: cmd/cmds/cmd_create.go
    - Path: cmd/cmds/cmd_delete.go
    - Path: cmd/cmds/cmd_diff.go
    - Path: cmd/cmds/cmd_discover.go
    - Path: cmd/cmds/cmd_fork.go
    - Path: cmd/cmds/cmd_info.go
    - Path: cmd/cmds/cmd_list.go
    - Path: cmd/cmds/cmd_merge.go
    - Path: cmd/cmds/cmd_rebase.go
    - Path: cmd/cmds/cmd_remove.go
    - Path: cmd/cmds/cmd_status.go
    - Path: pkg/wsm/workflows
      Note: Current orchestration layer to reuse from Glazed commands
ExternalSources:
    - /home/manuel/.codex/skills/glazed-command-authoring/SKILL.md
Summary: Command-by-command implementation plan to migrate all retained WSM CLI verbs from ad-hoc Cobra handlers to Glazed command definitions while preserving behavior.
LastUpdated: 2026-02-28T16:19:00-05:00
WhatFor: Provide an execution-ready migration blueprint for porting all CLI verbs to Glazed in phases.
WhenToUse: Use when implementing, reviewing, or sequencing the WSM Glazed migration.
---

# Glazed CLI Migration Implementation Plan

> Superseded by `design-doc/02-glazed-cli-migration-implementation-plan-v2.md` for layout, output mode, and section strategy decisions.

## Executive Summary

This document defines a complete migration plan for the current `wsm` command surface from hand-written Cobra `RunE` handlers to Glazed command definitions. The migration preserves existing behavior and output contracts while standardizing command schemas, settings decoding, and output middleware around Glazed conventions.

The guiding principle is: keep all business logic in `pkg/wsm` (primarily `pkg/wsm/workflows` and existing services), and make each CLI verb a thin Glazed command that decodes typed settings and delegates to reusable package APIs.

## Problem Statement

Current command code is functional but heterogeneous:

- every command defines flags manually through Cobra,
- command parsing, orchestration, and rendering style differs between files,
- output conventions and parser features are inconsistent,
- root command still relies on deprecated clay/logging bootstrapping.

This creates friction for:

- adding new verbs/subcommands quickly,
- generating/inspecting command schemas,
- maintaining consistent output controls across commands,
- onboarding engineers to one clear command authoring pattern.

## Proposed Solution

### Core migration approach

1. Introduce a Glazed command layer under `cmd/glazedcmds` (or `cmd/cmds/glazed`), one file per verb group.
2. For each verb/subcommand, define:
   - `type <Verb>Command struct { *cmds.CommandDescription }`
   - `type <Verb>Settings struct { ... `glazed:"..."` ... }`
   - `New<Verb>Command()` using `cmds.NewCommandDescription`, `fields.New`, and Glazed sections.
   - `RunIntoGlazeProcessor(...)` that decodes settings and delegates into `pkg/wsm` services/workflows.
3. Build Cobra adapters through `cli.BuildCobraCommandFromCommand(...)` and register these in root/group commands.
4. Migrate root bootstrapping to non-deprecated Glazed/clay logging init path.
5. Remove legacy Cobra `cmd/cmds/cmd_*.go` command implementations after parity is verified.

### Command architecture target

- `cmd/wsm/root.go`
  - Glazed-aware root parser config
  - grouped command registration
- `cmd/glazedcmds/<group>/root.go`
  - group-level registration
- `cmd/glazedcmds/<group>/<verb>.go`
  - command description, flags schema, settings decode, delegate call
- `pkg/wsm/...`
  - unchanged source of business logic (workflows/services)

### Canonical Glazed command shape (WSM-specific)

```go
type StatusCommand struct { *cmds.CommandDescription }

type StatusSettings struct {
    Workspace string `glazed:"workspace"`
    Short bool `glazed:"short"`
    Untracked bool `glazed:"untracked"`
    Jobs int `glazed:"jobs"`
}

func (c *StatusCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
    s := &StatusSettings{}
    if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
        return err
    }

    workflow := workflows.NewStatusWorkflow()
    st, err := workflow.GetStatus(ctx, workflows.StatusRequest{
        WorkspaceName: s.Workspace,
        Jobs: s.Jobs,
    })
    if err != nil {
        return err
    }

    // Either emit rows or call existing render helper and emit summary row.
    // Preferred: emit normalized rows and let output middleware handle format.
    return gp.AddRow(ctx, types.NewRow(
        types.MRP("workspace", st.Workspace.Name),
        types.MRP("overall", st.Overall),
    ))
}
```

## Design Decisions

1. **Preserve pkg workflow/service boundary**
- Do not re-embed business logic into Glazed commands.
- Glazed migration is a CLI layer refactor, not a domain logic rewrite.

2. **One migration pass per command group**
- Migrate grouped commands (`list`, `branch`, `rebase`) as cohesive units to avoid mixed parent/child wiring complexity.

3. **Schema-first flags**
- Use Glazed `fields.New(...)` as source of truth.
- Avoid direct Cobra flag reads in `Run` path.

4. **Explicit compatibility policy**
- No backward compatibility shims for already removed commands.
- Preserve behavior and flag names for retained commands unless explicitly changed in this ticket.

5. **Output strategy by command type**
- Data-oriented commands (`list`, `info`, `status`, `diff`, `log`, `branch list`) should emit rows.
- Action commands (`create`, `fork`, `commit`, `merge`, `delete`, `add`, `remove`, `rebase continue/abort`) may emit summary rows + human messages.

## Alternatives Considered

1. **Keep existing Cobra commands and add Glazed only for new verbs**
- Rejected: prolongs mixed command architecture and duplicates maintenance burden.

2. **Full rewrite including pkg workflows simultaneously**
- Rejected: too much risk; the pkg refactor was just completed and should be reused, not destabilized.

3. **Metadata-parent only approach (`cmds.WithParents`) without explicit Cobra groups**
- Not selected as default for WSM due current clear grouping and need for explicit phased migration.

## Command Inventory and Port Plan

The following sections cover every retained user-facing command/subcommand.

### Root Command (`wsm`)

**Current:** `cmd/wsm/root.go` with manual `AddCommand`, deprecated `clay.InitViper`, deprecated logger init.

**Glazed port plan:**
- Introduce Glazed parser config at root.
- Register group roots and leaf commands via Glazed-built Cobra commands.
- Replace deprecated init calls with current recommended Glazed/clay/logging setup.

**Key tasks:**
- Add root parser config with `ShortHelpSections` and middlewares.
- Keep existing global flags semantics.
- Ensure completion/help behavior still works.

---

### `discover`

**Current source:** `cmd/cmds/cmd_discover.go` + `workflows.DiscoverWorkflow`.

**Glazed settings:** `paths []string`, `recursive bool`, `max-depth int`.

**Port plan:**
- Create `DiscoverCommand` Glazed descriptor with positional paths support.
- Decode settings into `DiscoverRequest`, call workflow, emit summary row (`paths`, `repository_count`).
- Keep user-friendly informational messages but make row output the canonical machine interface.

**Risk:** low.

---

### `list` group

#### `list repos`

**Current source:** `cmd/cmds/cmd_list.go` + `workflows.ListWorkflow`.

**Settings:** `format`, `tags []string`.

**Port plan:**
- Emit one row per repository.
- Keep table-like pretty output via Glazed output options.
- Deprecate custom `--format` in favor of Glazed output options after parity phase (phase 2 cleanup).

#### `list workspaces`

**Settings:** `format`.

**Port plan:**
- Emit one row per workspace with key fields (`name`, `path`, `branch`, `repo_count`, `created`).
- Use sorted list returned by workflow.

**Risk:** low.

---

### `create`

**Current source:** `cmd/cmds/cmd_create.go` + `workflows.CreateWorkflow` + interactive repo selection.

**Settings:** `name`, `repos []string`, `branch`, `branch-prefix`, `base-branch`, `agent-source`, `interactive`, `dry-run`.

**Port plan:**
- Positional workspace name in command description.
- Decode settings then:
  - if `interactive`, perform prompt flow (can remain in command layer),
  - call `CreateWorkflow.Create`.
- Emit structured result row (`workspace`, `path`, `branch`, `repo_count`, `dry_run`).

**Risk:** medium due interactive path.

---

### `fork`

**Current source:** `cmd/cmds/cmd_fork.go` + `workflows.ForkWorkflow`.

**Settings:** `new-workspace-name`, optional source workspace positional, plus `branch`, `branch-prefix`, `agent-source`, `dry-run`, `workspace`.

**Port plan:**
- Build schema with one required positional + optional source positional and explicit `--workspace` override.
- Delegate planning + execution to workflow.
- Emit fork-plan row in dry-run and fork-result row in execute mode.

**Risk:** medium due dual source resolution paths.

---

### `merge`

**Current source:** `cmd/cmds/cmd_merge.go` + `workflows.MergeWorkflow`.

**Settings:** optional workspace positional, `dry-run`, `force`, `workspace`, `keep-workspace`.

**Port plan:**
- Minimal adapter around `MergeWorkflow.Execute`.
- For later enhancement: refactor workflow to optionally return row events for progress.

**Risk:** medium (interactive confirmation and rollback messaging).

---

### `add`

**Current source:** `cmd/cmds/cmd_add.go` direct `WorkspaceManager.AddRepositoryToWorkspace`.

**Settings:** positional `workspace-name`, positional `repo-name`, `branch`, `force`.

**Port plan:**
- Create `AddCommand` in Glazed with typed settings.
- Keep direct manager delegation initially; optional future extraction to workflow for consistency.

**Risk:** low.

---

### `remove`

**Current source:** `cmd/cmds/cmd_remove.go` direct `WorkspaceManager.RemoveRepositoryFromWorkspace`.

**Settings:** positional `workspace-name`, positional `repo-name`, `force`, `remove-files`.

**Port plan:**
- Mirror add-command strategy with Glazed schema and manager delegation.
- Emit summary row (`workspace`, `repo`, `removed_files`).

**Risk:** low.

---

### `delete`

**Current source:** `cmd/cmds/cmd_delete.go` + `workflows.DeleteWorkflow` + interactive confirm.

**Settings:** positional workspace, `force`, `force-worktrees`, `remove-files`, `output`.

**Port plan:**
- Use workflow `Preview` and `Delete`.
- Keep confirmation in command layer for interactive UX.
- Replace legacy `--output` with Glazed output options in cleanup phase.

**Risk:** medium due destructive action semantics.

---

### `info`

**Current source:** `cmd/cmds/cmd_info.go` + `workflows.InfoWorkflow`.

**Settings:** optional workspace positional, `workspace`, `output`, `field`.

**Port plan:**
- Decode settings and resolve workspace via workflow.
- If `field` is set, emit single-field row/value output.
- Remove custom format branching after parity phase.

**Risk:** low.

---

### `status`

**Current source:** `cmd/cmds/cmd_status.go` + `workflows.StatusWorkflow`.

**Settings:** optional workspace positional, `workspace`, `short`, `untracked`, `jobs`.

**Port plan:**
- Delegate status retrieval to workflow.
- Emit one row per repository with standardized columns.
- Preserve short/detailed modes initially, then collapse into output transforms.

**Risk:** medium due current rich textual formatting.

---

### `commit`

**Current source:** `cmd/cmds/cmd_commit.go` + `workflows.CommitWorkflow`.

**Settings:** `message`, `interactive`, `add-all`, `push`, `dry-run`, `template`.

**Port plan:**
- Decode into `CommitRequest` and call `Prepare`.
- Keep interactive selection in command layer initially.
- Call workflow `Execute` and emit summary row (`repo_count`, `pushed`, `dry_run`).

**Risk:** medium due interactive selection and messaging.

---

### `branch` group

#### `branch create`

**Current source:** `cmd/cmds/cmd_branch.go` + `BranchOperations`.

**Settings:** optional `branch-name`, `track`.

**Port plan:**
- Glazed subcommand with positional branch name and `track` bool.
- Emit per-repo result rows from operation result DTO.

#### `branch switch`

**Settings:** optional `branch-name`.

**Port plan:**
- Same as create, using `SwitchBranch` operation.

#### `branch list`

**Settings:** none.

**Port plan:**
- Emit branches per repo as rows (repo, branch, current, tracking).

**Risk:** low-medium.

---

### `rebase` group

#### `rebase`

**Current source:** `cmd/cmds/cmd_rebase.go` + `workflows.RebaseWorkflow`.

**Settings:** optional repo positional, `target`, `dry-run`, `interactive`, `jobs`, `manual`.

**Port plan:**
- Decode into `RebaseRequest`; support manual command-plan mode.
- Emit result rows for each repo (`success`, `conflicts`, `rebased`).

#### `rebase status`

**Settings:** `repo`, `jobs`.

**Port plan:**
- Call workflow `Status`; emit rows directly.

#### `rebase continue`

**Settings:** `repo`, `jobs`.

**Port plan:**
- Call workflow `Continue`; emit action rows.

#### `rebase abort`

**Settings:** `repo`, `jobs`.

**Port plan:**
- Call workflow `Abort`; emit action rows.

**Risk:** medium-high due concurrency and mixed interactive/manual modes.

---

### `diff`

**Current source:** `cmd/cmds/cmd_diff.go` + `GitOperations.GetDiff`.

**Settings:** `staged`, `repo`, `jobs`.

**Port plan:**
- Wrap current diff text output as either raw field (`diff`) rows per repo or streamed text row.
- Consider introducing repo-wise diff workflow for better row-level output.

**Risk:** medium (non-tabular content).

---

### `log`

**Current source:** `cmd/cmds/cmd_diff.go` + `HistoryOperations.GetWorkspaceLog`.

**Settings:** `since`, `oneline`, `limit`.

**Port plan:**
- Emit commit rows (repo, hash, author, date, subject) when possible.
- Maintain one-line compact mode with row projection rather than bespoke format.

**Risk:** low-medium.

## Cross-Cutting Migration Notes

### Completion integrations (Carapace)

Current commands use Carapace completions heavily. Glazed migration should keep existing completion behavior in phase 1 by attaching completion callbacks to generated Cobra commands. Completion modernization can be deferred.

### Output normalization

Several commands still have custom `--format` flags. During parity phase keep those flags, then remove/alias them in a cleanup phase after users move to Glazed `--output` options.

### Error semantics

Preserve current error texts for destructive or high-signal flows (`create`, `delete`, `merge`, `rebase`) until dedicated UX cleanup ticket.

## Implementation Plan

### Phase 0: Foundation

1. Add `cmd/glazedcmds` package scaffold and group roots.
2. Add Glazed helper utilities for positional arg decoding and result-row helpers.
3. Update root wiring to support Glazed command registration.

### Phase 1: Low-risk verbs

Migrate:
- `discover`
- `list repos`
- `list workspaces`
- `info`
- `add`
- `remove`

Validation:
- golden snapshots for `--help` and output shape.

### Phase 2: Medium verbs

Migrate:
- `status`
- `diff`
- `log`
- `branch create/switch/list`

Validation:
- repo fixture tests for status/branch/log consistency.

### Phase 3: Workflow-heavy verbs

Migrate:
- `create`
- `fork`
- `commit`
- `delete`

Validation:
- integration tests for interactive and non-interactive paths.

### Phase 4: High-risk orchestration verbs

Migrate:
- `rebase` (+status/continue/abort)
- `merge`

Validation:
- conflict scenarios, rollback semantics, jobs concurrency.

### Phase 5: Cleanup and cutover

1. Remove legacy Cobra command files under `cmd/cmds/cmd_*.go`.
2. Remove deprecated init/logging root code paths.
3. Update help docs and implementation docs.
4. Final doctor check and publish migration diary.

## Detailed Task Breakdown by Command

- `wsm root`: replace deprecated init APIs, wire Glazed parser/middlewares.
- `discover`: define schema + delegate to `DiscoverWorkflow`.
- `list`: split into `repos` and `workspaces` glazed subcommands.
- `create`: keep interactive repo prompt, delegate execution to `CreateWorkflow`.
- `fork`: preserve source detection logic through `ForkWorkflow.Plan/Fork`.
- `merge`: keep workflow execute semantics, migrate flag schema.
- `add`: direct manager delegation via glazed settings.
- `remove`: direct manager delegation via glazed settings.
- `delete`: preserve confirm behavior, move parsing/output to glazed.
- `info`: migrate field extraction and workspace selection to glazed settings.
- `status`: expose row-oriented output from current status data.
- `commit`: preserve interactive selection, use `CommitWorkflow.Prepare/Execute`.
- `branch create`: migrate to glazed subcommand with result rows.
- `branch switch`: migrate to glazed subcommand with result rows.
- `branch list`: row output by repo/branch.
- `rebase`: migrate main and action subcommands with typed settings.
- `diff`: map text diff payload into glazed output strategy.
- `log`: row-based commit projection and compact output option.

## Testing and Validation Plan

- Unit tests per new Glazed command settings decode.
- Existing `pkg/wsm` tests unchanged and reused.
- Command parity checks:
  - `wsm <cmd> --help` snapshots before/after.
  - fixture-based output comparisons for core commands.
- Integration pass:
  - `go test ./cmd/... ./pkg/...`
  - targeted `test/integration/scenarios` once fixture issues are isolated.

## Open Questions

1. Should we keep custom `--format` flags long-term or fully drop them in favor of Glazed `--output`?
2. For text-heavy commands (`diff`, some `rebase` outputs), do we emit structured rows only, text only, or dual mode?
3. Should Carapace completions remain, or should completion be standardized around Glazed-generated metadata?

## References

- Glazed skill guide: `/home/manuel/.codex/skills/glazed-command-authoring/SKILL.md`
- Current root wiring: `/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/root.go`
- Current command implementations: `/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds`
- Current orchestration services: `/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workflows`
