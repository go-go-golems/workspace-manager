---
Title: WSM Glazed Dual Output Implementation Plan
Ticket: WSM-MO-009-GLAZED-DUAL-OUTPUT-WSM
Status: active
Topics:
    - architecture
    - glazed
    - refactor
    - workspace-manager
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/glazed/pkg/doc/tutorials/05-build-first-command.md
      Note: Canonical RunIntoGlazeProcessor guidance
    - Path: ../../../../../../../glazed/cmd/examples/new-api-dual-mode/main.go
      Note: Reference dual-mode command implementation
    - Path: cmd/wsm/cmds/common/build.go
      Note: |-
        Current Cobra builder wrapper; needs dual-mode wiring
        Builder wrapper to migrate to native dual-mode options
        Added transitional dual-mode builder used by migrated registry commands
    - Path: cmd/wsm/cmds/common/runtime.go
      Note: |-
        Current custom dual-output helper (output-mode + EmitRows) to be replaced by GlazeCommand pattern
        Custom EmitRows/output-mode path to deprecate
    - Path: cmd/wsm/cmds/git
      Note: Git command set migration scope
    - Path: cmd/wsm/cmds/git/branch.go
      Note: Grouped command file to split into git/branch/* structure
    - Path: cmd/wsm/cmds/git/commit.go
      Note: Representative interactive git command migration target
    - Path: cmd/wsm/cmds/git/rebase.go
      Note: Grouped command file to split into git/rebase/* structure
    - Path: cmd/wsm/cmds/js/runner.go
      Note: JS runner command migration scope
    - Path: cmd/wsm/cmds/registry
      Note: Registry command set migration scope
    - Path: cmd/wsm/cmds/registry/discover.go
      Note: |-
        Representative simple command migration target
        Migrated to Run + RunIntoGlazeProcessor with execute helper
    - Path: cmd/wsm/cmds/registry/list_repos.go
      Note: |-
        Concrete target for concise human-template output refinement
        Migrated to Run + RunIntoGlazeProcessor while keeping concise human template
    - Path: cmd/wsm/cmds/registry/list_workspaces.go
      Note: |-
        Concrete target for concise human-template output refinement
        Migrated to Run + RunIntoGlazeProcessor while keeping concise human template
    - Path: cmd/wsm/cmds/registry/root.go
      Note: Current list parent registration to move under registry/list/root.go
    - Path: cmd/wsm/cmds/workspace
      Note: Workspace command set migration scope
    - Path: cmd/wsm/cmds/workspace/create.go
      Note: Representative interactive workspace command migration target
ExternalSources:
    - /home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/tutorials/05-build-first-command.md
    - /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/glazed/cmd/examples/new-api-dual-mode/main.go
Summary: Migration plan to make every WSM command use native Glazed dual-mode (Run for human, RunIntoGlazeProcessor for structured output) and remove custom EmitRows/output-mode plumbing.
LastUpdated: 2026-03-01T11:10:00-05:00
WhatFor: Provide an implementation-ready command-by-command plan for unifying WSM around the Glazed dual-output pattern.
WhenToUse: Use this document when implementing or reviewing WSM CLI command migrations away from EmitRows to RunIntoGlazeProcessor.
---




# WSM Glazed Dual Output Implementation Plan

## Executive Summary

This plan migrates all `wsm` commands to the native Glazed dual-output model:

1. `Run(...)` is human-only output.
2. `RunIntoGlazeProcessor(...)` is structured output only (`gp.AddRow(...)`).
3. The custom `EmitRows(...)` helper path is removed.

Today, all command adapters are `cmds.BareCommand` only and manually gate output with `--output-mode` plus `EmitRows` in command bodies (`cmd/wsm/cmds/*`). The target state is the Glazed tutorial pattern (`GlazeCommand` + `RunIntoGlazeProcessor`) with dual-mode Cobra wiring.

## Problem Statement

Current WSM command output handling is not leveraging Glazed end-to-end.

Observed evidence:

1. Every command adapter currently declares `cmds.BareCommand` and none declare `cmds.GlazeCommand` (`cmd/wsm/cmds/...`, e.g. `cmd/wsm/cmds/git/commit.go:35`, `cmd/wsm/cmds/workspace/create.go:38`, `cmd/wsm/cmds/registry/discover.go:31`, `cmd/wsm/cmds/js/runner.go:31`).
2. Structured output is emitted manually via `wsmcmdcommon.EmitRows(...)` in all command groups (`cmd/wsm/cmds/git/*.go`, `cmd/wsm/cmds/workspace/*.go`, `cmd/wsm/cmds/registry/*.go`, `cmd/wsm/cmds/js/runner.go`).
3. `EmitRows` centralizes processor setup in `cmd/wsm/cmds/common/runtime.go:94-123`, which bypasses the direct `RunIntoGlazeProcessor` pattern from Glazed docs.
4. Output behavior is governed by a custom runtime section (`output-mode=human|data|both`) in `cmd/wsm/cmds/common/runtime.go:18-45`.

From Glazed docs/examples:

1. Build-first-command tutorial states structured output should be implemented in `RunIntoGlazeProcessor` (`.../05-build-first-command.md:103-139`).
2. Dual-mode example shows one command implementing both `Run` and `RunIntoGlazeProcessor`, with Cobra dual-mode wiring (`glazed/cmd/examples/new-api-dual-mode/main.go:49-104`).
3. CLI supports native dual-mode toggles (`glazed/pkg/cli/cobra.go:474-486` and run-path selection at `:165-215`).

## Scope

In scope:

1. All `wsm` command verbs under `cmd/wsm/cmds/{registry,workspace,git,js}`.
2. Common command wiring in `cmd/wsm/cmds/common/build.go` and `cmd/wsm/cmds/common/runtime.go`.
3. Human-output behavior consistency in `Run` for every command.
4. Directory normalization so grouped verbs mirror CLI structure:
- `cmd/wsm/cmds/git/branch/{root.go,create.go,switch.go,list.go}`
- `cmd/wsm/cmds/git/rebase/{root.go,rebase.go,status.go,continue.go,abort.go}`
- `cmd/wsm/cmds/registry/list/{root.go,repos.go,workspaces.go}`

Out of scope:

1. Rewriting domain workflows in `pkg/wsm/workflows` unless required by command adapter splitting.
2. Renaming end-user CLI verbs.

## Target Design

## Output contract

For every command type `XCommand`:

1. `Run(ctx, vals)`:
- Produces human-readable output only.
- Handles prompts/interactive UX.
- Must not call `gp.AddRow` or any structured-output helper.

2. `RunIntoGlazeProcessor(ctx, vals, gp)`:
- Produces structured rows only.
- Uses `gp.AddRow(ctx, row)` directly.
- Must not print to stdout/stderr (except errors).

3. Interface declarations:
- `var _ cmds.BareCommand = &XCommand{}`
- `var _ cmds.GlazeCommand = &XCommand{}`

## Cobra wiring

Update `cmd/wsm/cmds/common/build.go` to enable dual-mode routing through Glazed CLI builder options:

```go
func BuildCobraCommand(command cmds.Command) (*cobra.Command, error) {
    return cli.BuildCobraCommandFromCommand(command,
        cli.WithDualMode(true),
        cli.WithGlazeToggleFlag("with-glaze-output"),
        cli.WithParserConfig(cli.CobraParserConfig{
            ShortHelpSections: []string{schema.DefaultSlug},
            MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
        }),
    )
}
```

## Runtime helper simplification

1. Remove `EmitRows(...)` from `cmd/wsm/cmds/common/runtime.go`.
2. Remove custom `output-mode` field and resolver helpers once all commands are migrated.
3. Keep only shared utilities that are still needed (if any).

## Directory and registrar structure

Grouped command families should be physically grouped to mirror verb structure and make parent wiring local:

1. `git branch` family:
- current: single file `cmd/wsm/cmds/git/branch.go` with parent + children.
- target:
  - `cmd/wsm/cmds/git/branch/root.go` (`Use: branch`, child registration)
  - `cmd/wsm/cmds/git/branch/create.go`
  - `cmd/wsm/cmds/git/branch/switch.go`
  - `cmd/wsm/cmds/git/branch/list.go`

2. `git rebase` family:
- current: single file `cmd/wsm/cmds/git/rebase.go`.
- target:
  - `cmd/wsm/cmds/git/rebase/root.go`
  - `cmd/wsm/cmds/git/rebase/rebase.go`
  - `cmd/wsm/cmds/git/rebase/status.go`
  - `cmd/wsm/cmds/git/rebase/continue.go`
  - `cmd/wsm/cmds/git/rebase/abort.go`

3. `registry list` family:
- current: parent in `cmd/wsm/cmds/registry/root.go` and leaf files at top level.
- target:
  - `cmd/wsm/cmds/registry/list/root.go`
  - `cmd/wsm/cmds/registry/list/repos.go`
  - `cmd/wsm/cmds/registry/list/workspaces.go`

This preserves CLI command names while making filesystem layout and runtime verb hierarchy consistent.

## Command implementation pattern

Use shared execution helpers to avoid duplicating core logic between `Run` and `RunIntoGlazeProcessor`:

```go
type StatusResult struct { ... }

func (c *StatusCommand) execute(ctx context.Context, vals *values.Values) (*StatusResult, error) {
    // decode settings + call workflows
}

func (c *StatusCommand) Run(ctx context.Context, vals *values.Values) error {
    result, err := c.execute(ctx, vals)
    if err != nil { return err }
    return renderStatusHuman(result)
}

func (c *StatusCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
    result, err := c.execute(ctx, vals)
    if err != nil { return err }
    for _, row := range resultToRows(result) {
        if err := gp.AddRow(ctx, row); err != nil { return err }
    }
    return nil
}
```

## Command-by-command migration matrix

## Registry commands

1. `wsm discover` (`cmd/wsm/cmds/registry/discover.go`)
- Current: BareCommand + `ResolveOutputMode` + `EmitRows` (`:73-117`, `:102-109`).
- Change: keep current workflow call; split renderer into `Run` (informational lines) and `RunIntoGlazeProcessor` (single summary row: `paths`, `repository_count`).

2. `wsm list repos` (`cmd/wsm/cmds/registry/list_repos.go`)
- Current: human table via tabwriter + row slice + `EmitRows` (`:54-98`, `:100-141`).
- Change: replace tabwriter with a concise human template in `Run` (fast visual scan, no aligned table formatting); move row emission loop to `RunIntoGlazeProcessor`.

3. `wsm list workspaces` (`cmd/wsm/cmds/registry/list_workspaces.go`)
- Current: same mixed pattern (`:40-87`, `:89-125`).
- Change: replace tabwriter with a concise human template in `Run` that quickly communicates workspace identity/status/path; keep structured rows in `RunIntoGlazeProcessor`.

## Workspace commands

4. `wsm create` (`cmd/wsm/cmds/workspace/create.go`)
- Current: mixed output-mode path with `EmitRows` (`:65-175`).
- Change: `Run` keeps preview/success UX and interactive selection; `RunIntoGlazeProcessor` emits deterministic result rows and rejects interactive prompts with a clear error if required.

5. `wsm fork` (`cmd/wsm/cmds/workspace/fork.go`)
- Current: mixed path with plan+fork flow and `EmitRows` (`:60-174`).
- Change: `Run` keeps preview/details text; `RunIntoGlazeProcessor` emits plan/result rows only.

6. `wsm merge` (`cmd/wsm/cmds/workspace/merge.go`)
- Current: has structured output but effectively no human summary path beyond workflow side effects (`:53-93`).
- Change: add explicit human result summary in `Run`; structured summary row in `RunIntoGlazeProcessor`.

7. `wsm add` (`cmd/wsm/cmds/workspace/add.go`)
- Current: only structured row path is explicit (`:89-139`).
- Change: add human success summary in `Run`; row emission in `RunIntoGlazeProcessor`.

8. `wsm remove` (`cmd/wsm/cmds/workspace/remove.go`)
- Current: same as add (`:89-139`).
- Change: add human success summary in `Run`; row emission in `RunIntoGlazeProcessor`.

9. `wsm delete` (`cmd/wsm/cmds/workspace/delete.go`)
- Current: interactive confirmation + preview + `EmitRows` in single flow (`:56-177`).
- Change: keep confirmations/human warnings in `Run`; in `RunIntoGlazeProcessor`, require non-interactive usage (`--force`) and emit preview/decision/result rows.

10. `wsm info` (`cmd/wsm/cmds/workspace/info.go`)
- Current: field mode and full mode both mixed with `EmitRows` (`:70-145`).
- Change: `Run` prints field or detailed summary; `RunIntoGlazeProcessor` emits either field row or workspace row.

11. `wsm status` (`cmd/wsm/cmds/workspace/status.go`)
- Current: large mixed flow with human short/detailed format and data rows (`:83-156`).
- Change: keep short/detailed human renderers in `Run`; move row projection to `RunIntoGlazeProcessor`; reuse one shared compute function.

## Git commands

12. `wsm commit` (`cmd/wsm/cmds/git/commit.go`)
- Current: mixed flow, interactive prompt, multiple early data emissions (`:58-184`, interactive helper at `:186-219`).
- Change: `Run` handles interactive commit UX and human statuses; `RunIntoGlazeProcessor` handles non-interactive execution only and emits summary + per-repo rows.

13. `wsm diff` (`cmd/wsm/cmds/git/diff.go`)
- Current: mixed mode with monolithic diff row (`:65-122`).
- Change: `Run` keeps readable diff output; `RunIntoGlazeProcessor` emits structured diff rows only.

14. `wsm log` (`cmd/wsm/cmds/git/log.go`)
- Current: mixed mode with repo log map projection (`:66-161`).
- Change: `Run` keeps grouped log rendering; `RunIntoGlazeProcessor` emits one row per repo log entry.

15. `wsm branch create` (`cmd/wsm/cmds/git/branch/create.go` target; currently in `cmd/wsm/cmds/git/branch.go`)
- Current: mixed mode (`:77-128`).
- Change: human summary table stays in `Run`; structured per-repo result rows in `RunIntoGlazeProcessor`.

16. `wsm branch switch` (`cmd/wsm/cmds/git/branch/switch.go` target; currently in `cmd/wsm/cmds/git/branch.go`)
- Current: mixed mode (`:150-200`).
- Change: same split as branch create.

17. `wsm branch list` (`cmd/wsm/cmds/git/branch/list.go` target; currently in `cmd/wsm/cmds/git/branch.go`)
- Current: mixed mode with tabwriter and row collection in same function (`:214-301`).
- Change: keep table output in `Run`; move row conversion to `RunIntoGlazeProcessor`.

18. `wsm rebase` (`cmd/wsm/cmds/git/rebase/rebase.go` target; currently in `cmd/wsm/cmds/git/rebase.go`)
- Current: mixed mode with manual plan and result rendering (`:83-178`).
- Change: human instructions/tables in `Run`; structured rebase plan/result rows in `RunIntoGlazeProcessor`.

19. `wsm rebase status` (`cmd/wsm/cmds/git/rebase/status.go` target; currently in `cmd/wsm/cmds/git/rebase.go`)
- Current: mixed mode (`:195-241`).
- Change: human table in `Run`; structured state rows in `RunIntoGlazeProcessor`.

20. `wsm rebase continue` (`cmd/wsm/cmds/git/rebase/continue.go` target; currently in `cmd/wsm/cmds/git/rebase.go`)
- Current: shared mixed helper `runRebaseAction` (`:281-335`).
- Change: split helper into human and structured projections; keep command entrypoint thin.

21. `wsm rebase abort` (`cmd/wsm/cmds/git/rebase/abort.go` target; currently in `cmd/wsm/cmds/git/rebase.go`)
- Current: same shared path as continue.
- Change: same as continue.

## JS command

22. `wsm runner` (`cmd/wsm/cmds/js/runner.go`)
- Current: mixed mode with JSON printing + `EmitRows` (`:62-106`).
- Change: `Run` prints execution summary and optional result pretty-print; `RunIntoGlazeProcessor` emits script/result rows only.

## Group wrapper commands (structural, not output-bearing)

1. `wsm list` parent should move to `cmd/wsm/cmds/registry/list/root.go` (currently `cmd/wsm/cmds/registry/root.go:25-33`).
2. `wsm branch` parent should move to `cmd/wsm/cmds/git/branch/root.go` (currently `cmd/wsm/cmds/git/branch.go:349-384`).
3. `wsm rebase` parent/subcommand composition should move to `cmd/wsm/cmds/git/rebase/root.go` (currently `cmd/wsm/cmds/git/rebase.go:454-490`).

These wrappers stay, but child commands should all support native dual-mode behavior.

## Implementation Plan

## Phase 1: Framework plumbing

1. Create grouped subdirectories and local `root.go` registrars (`git/branch`, `git/rebase`, `registry/list`) without changing CLI surface.
2. Move existing grouped implementations into the new layout first (no behavior changes).
3. Update `cmd/wsm/cmds/common/build.go` to enable dual-mode builder options.
4. Pick and document the toggle flag (`--with-glaze-output` default, optionally alias old behavior).
5. Add one migrated exemplar command (recommended: `registry discover`) to validate pattern.

## Phase 2: Remove custom runtime output path

1. Stop adding runtime `output-mode` in new command descriptions.
2. Migrate command files from mode checks + `EmitRows` to `RunIntoGlazeProcessor`.
3. Delete `EmitRows`, `ResolveOutputMode`, `ShouldOutput*`, and runtime `output-mode` definitions from `cmd/wsm/cmds/common/runtime.go` after last command migration.

## Phase 3: Command group migration sweep

1. Registry + JS (low complexity).
2. Workspace (medium complexity, includes prompts and preview output).
3. Git (highest complexity, especially commit and rebase families).

## Phase 4: Consistency and UX hardening

1. Ensure every command has explicit human summary output in `Run`.
2. Ensure structured mode never prints narrative text.
3. Add cross-command guidance in help text for human vs structured execution.

## Test Strategy

## Unit-level assertions

1. For each command, test `Run` path emits expected human behavior (golden stdout where practical).
2. For each command, test `RunIntoGlazeProcessor` emits expected row schema/values.
3. For interactive commands, verify structured mode rejects prompt-required flows with explicit errors.

## Integration-level assertions

1. CLI classic mode: `wsm <command>` continues to produce human output.
2. CLI structured mode: `wsm <command> --with-glaze-output --output json` returns machine-parseable rows only.
3. No command still references `EmitRows` after migration (`rg -n "EmitRows\(" cmd/wsm/cmds`).

## Risks, Alternatives, Open Questions

## Risks

1. Behavioral drift for scripts currently using `--output-mode`.
2. Interactive commands (`create`, `delete`, `commit`) need clear structured-mode policy.
3. Large output commands (`diff`, `log`) may require row-shape tuning for downstream consumers.

## Alternatives considered

1. Keep `output-mode` + `EmitRows` as-is.
- Rejected: duplicates Glazed plumbing and keeps structured output logic scattered.

2. Switch to Glaze-only commands (remove human `Run`).
- Rejected: would break human-first CLI UX and interactive command flows.

## Open questions

1. Backward compatibility window for `--output-mode`: deprecate immediately or support for one release?
2. Toggle convention: retain `--with-glaze-output` default from Glazed, or define a WSM alias?
3. Should long-text payloads (`diff`, repo `log`) be split into per-file/per-commit rows in structured mode?

## References

1. `cmd/wsm/cmds/common/runtime.go:18-138`
2. `cmd/wsm/cmds/common/build.go:10-17`
3. `cmd/wsm/cmds/registry/discover.go:73-117`
4. `cmd/wsm/cmds/registry/list_repos.go:54-98`
5. `cmd/wsm/cmds/registry/list_workspaces.go:40-87`
6. `cmd/wsm/cmds/workspace/create.go:65-175`
7. `cmd/wsm/cmds/workspace/fork.go:60-174`
8. `cmd/wsm/cmds/workspace/delete.go:56-177`
9. `cmd/wsm/cmds/workspace/status.go:83-156`
10. `cmd/wsm/cmds/workspace/info.go:70-145`
11. `cmd/wsm/cmds/workspace/add.go:89-139`
12. `cmd/wsm/cmds/workspace/remove.go:89-139`
13. `cmd/wsm/cmds/workspace/merge.go:53-93`
14. `cmd/wsm/cmds/git/commit.go:58-184`
15. `cmd/wsm/cmds/git/diff.go:65-122`
16. `cmd/wsm/cmds/git/log.go:66-161`
17. `cmd/wsm/cmds/git/branch.go:77-301`
18. `cmd/wsm/cmds/git/rebase.go:83-335`
19. `cmd/wsm/cmds/js/runner.go:62-106`
20. `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/tutorials/05-build-first-command.md:103-139`
21. `/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/glazed/cmd/examples/new-api-dual-mode/main.go:49-104`
22. `/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/glazed/pkg/cli/cobra.go:165-215`
