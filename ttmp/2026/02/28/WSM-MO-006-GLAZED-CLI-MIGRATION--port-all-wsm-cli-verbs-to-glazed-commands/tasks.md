# Tasks

## Phase 0: Ticket Setup and Baseline Inventory (Completed)

- [x] Create ticket workspace and base docs
- [x] Inventory all retained CLI verbs and subcommands from `cmd/wsm/root.go` and `cmd/cmds/*`
- [x] Capture flag surfaces and grouping relationships (`list`, `branch`, `rebase`)
- [x] Identify current `pkg/wsm/workflows` delegation points to preserve
- [x] Publish v2 design direction (layout + output policy + section strategy)

## Phase 1: Glazed Migration Foundation

- [x] Create `cmd/wsm/cmds/common` package for shared Glazed command construction helpers
- [x] Create group directories and roots:
- [x] `cmd/wsm/cmds/registry/root.go`
- [x] `cmd/wsm/cmds/workspace/root.go`
- [x] `cmd/wsm/cmds/git/root.go` (placeholder for later phases)
- [x] Add shared dual-output helper (human-first default, structured opt-in)
- [x] Add shared parser config helper (`ShortHelpSections`, default middlewares)
- [x] Add minimal reusable runtime section helper (Option B baseline)
- [x] Update `cmd/wsm/root.go` imports to consume new group roots while legacy commands still coexist
- [x] Replace deprecated `clay.InitViper` with `clay.InitGlazed`
- [x] Replace deprecated `logging.InitLoggerFromViper` with `logging.InitLoggerFromCobra`
- [x] Verify root help and startup no longer prints deprecation warnings

## Phase 2: Low-Risk Verb Migration

- [x] Migrate `discover` to `cmd/wsm/cmds/registry/discover.go`
- [x] `discover`: implement human output path (`BareCommand.Run`)
- [x] `discover`: implement structured row output path (`GlazeCommand.RunIntoGlazeProcessor`)
- [x] Migrate `list repos` to `cmd/wsm/cmds/registry/list_repos.go`
- [x] `list repos`: implement human output path
- [x] `list repos`: implement structured row output path
- [x] Migrate `list workspaces` to `cmd/wsm/cmds/registry/list_workspaces.go`
- [x] `list workspaces`: implement human output path
- [x] `list workspaces`: implement structured row output path
- [x] Migrate `info` to `cmd/wsm/cmds/workspace/info.go`
- [x] `info`: implement human output path
- [x] `info`: implement structured row output path
- [x] Migrate `add` to `cmd/wsm/cmds/workspace/add.go`
- [x] `add`: implement human output path
- [x] `add`: implement structured row output path
- [x] Migrate `remove` to `cmd/wsm/cmds/workspace/remove.go`
- [x] `remove`: implement human output path
- [x] `remove`: implement structured row output path
- [x] Register migrated low-risk commands via new group roots in `cmd/wsm/root.go`
- [x] Keep legacy commands registered only for verbs not yet migrated
- [x] Add targeted tests for low-risk command setting decode / structured rows
- [x] Validate low-risk command parity via manual smoke checks

## Phase 3: Medium Verb Migration

- [x] Migrate `status`
- [x] Migrate `diff`
- [x] Migrate `log`
- [x] Migrate `branch create`
- [x] Migrate `branch switch`
- [x] Migrate `branch list`
- [x] Add parity tests for status/branch/diff/log behavior

## Phase 4: Workflow-Heavy Verb Migration

- [x] Migrate `create`
- [x] Migrate `fork`
- [x] Migrate `commit`
- [x] Migrate `delete`
- [x] Ensure interactive paths still function for `create`, `commit`, `delete`
- [x] Add fixture-driven tests for workflow-heavy verbs

## Phase 5: High-Risk Orchestration Migration

- [x] Migrate `rebase`
- [x] Migrate `rebase status`
- [x] Migrate `rebase continue`
- [x] Migrate `rebase abort`
- [x] Migrate `merge`
- [x] Validate conflict/rollback and concurrency semantics remain unchanged

## Phase 6: Cleanup and Finalization

- [x] Remove legacy `cmd/cmds` command implementations after parity confirmation
- [x] Normalize output strategy around `--output-mode human|data|both` and Glazed data output controls
- [x] Update README/help/implementation docs for Glazed architecture
- [x] Run `go test ./cmd/... ./pkg/...` and targeted integration scenarios
- [x] Run `docmgr doctor --ticket WSM-MO-006-GLAZED-CLI-MIGRATION --stale-after 30`
- [x] Upload implementation plan to reMarkable
- [x] Upload v2 implementation plan to reMarkable
- [x] Upload diary to reMarkable

## Immediate Execution Batch (Current Work Session)

- [x] Complete Phase 1 foundation in code
- [x] Complete Phase 2 low-risk migration in code
- [x] Commit foundation batch
- [x] Commit low-risk migration batch
- [x] Record detailed diary entries for this session
