# Changelog

## 2026-02-28

- Initialized ticket `WSM-MO-006-GLAZED-CLI-MIGRATION`.
- Added full command-by-command Glazed migration implementation plan.
- Added phased task breakdown for migration execution.
- Added detailed investigation diary entries covering setup, inventory, and planning.
- Uploaded implementation plan to reMarkable path `/ai/2026/02/28/WSM-MO-006-GLAZED-CLI-MIGRATION`.
- Added `glazed` topic to docmgr vocabulary to satisfy doctor checks.
- Added v2 implementation plan with finalized layout `cmd/wsm/cmds/<group>/<verb>.go`, human-first dual output policy, and minimal section strategy.
- Implemented Phase 1 scaffold and partial Phase 2 migration in code (commit `295ec27`): Glazed root wiring, common runtime/parser helpers, and migrated `discover`, `list repos`, `list workspaces`, `info`.
- Implemented remaining low-risk migration for `add` and `remove` (commit `b18b342`).
- Verified root/help startup works without deprecated init warnings and validated command package tests with `go test ./cmd/wsm/... ./pkg/... -count=1`.
- Implemented first Phase 3 medium command migration (commit `e879879`): `workspace/status` moved to Glazed command with dual output and legacy-human renderer parity.
- Implemented additional Phase 3 medium command migrations (commit `e57bc54`): moved `git/diff` and `git/log` to Glazed commands and switched root registration away from legacy handlers.
- Implemented remaining planned Phase 3 command migrations for branch operations (commit `0f3b95e`): moved `branch create`, `branch switch`, and `branch list` to Glazed subcommands under git group.
- Implemented first Phase 4 workflow-heavy command migration (commit `a08684b`): moved `commit` to a Glazed command while preserving interactive selection flow.
- Implemented remaining workflow/high-risk command migrations (commit `a40fc6c`): moved `workspace/create`, `workspace/fork`, `workspace/delete`, and `workspace/merge` to Glazed commands.
- Implemented full rebase command migration (commit `bad17a2`): moved `rebase`, `rebase status`, `rebase continue`, and `rebase abort` to Glazed command hierarchy.
- Completed cleanup cutover (commit `229bb14`): removed legacy `cmd/cmds` command implementations and updated top-level docs to the new command layout.
- Expanded investigation diary with Step 11-13 to document the final migration wave (`a40fc6c`, `bad17a2`, `229bb14`) including validation details and known integration-suite caveat.
- Re-ran ticket hygiene check: `docmgr doctor --ticket WSM-MO-006-GLAZED-CLI-MIGRATION --stale-after 30` reports all checks passing.
- Uploaded updated investigation diary to reMarkable path `/ai/2026/02/28/WSM-MO-006-GLAZED-CLI-MIGRATION`.
- Added integration harness hardening and parity coverage tests (commit `f3d459d`):
  - sandbox env now pins `XDG_CONFIG_HOME`/`XDG_CACHE_HOME`/`XDG_STATE_HOME` and non-interactive git editor settings,
  - test runner now builds/uses a sandbox-local binary by default to avoid stale `.out/wsm`,
  - new scenarios cover low-risk data output, branch/log human+data parity, workflow-heavy create/commit/delete data output, and focused concurrency/conflict validation runs.
- Validation update:
  - targeted matrix passes:
    - `go test ./test/integration/scenarios -run 'TestLowRiskCommandsDataOutput|TestBranchAndLogHumanDataParity|TestWorkflowHeavyCommandsDataOutput|TestSmokeStatusDiff|TestWorktreeCreateDelete|TestJobsConcurrency|TestRebaseConflictsAbort|TestRebaseConflictsContinueAbort' -count=1`
  - full suite still has known non-migration failures:
    - `TestCommitPush` (`push: remote not found` in hybrid backend)
    - `TestRebaseHappyPath`/`TestSyncAheadBehind` (legacy `sync` command removed)
- Validation snapshot:
  - `go test ./cmd/... ./pkg/... -count=1` passes
  - `go test ./... -count=1` still fails in integration scenarios with pre-existing sandbox/config isolation issue (`discover` sees host registry and `create` fails opening repo worktrees)
  - `go run ./cmd/wsm --help` shows all verbs served from migrated command groups.

### Related Files

- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/ttmp/2026/02/28/WSM-MO-006-GLAZED-CLI-MIGRATION--port-all-wsm-cli-verbs-to-glazed-commands/design-doc/01-glazed-cli-migration-implementation-plan.md — Primary implementation plan
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/ttmp/2026/02/28/WSM-MO-006-GLAZED-CLI-MIGRATION--port-all-wsm-cli-verbs-to-glazed-commands/design-doc/02-glazed-cli-migration-implementation-plan-v2.md — v2 implementation plan with updated architecture decisions
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/ttmp/2026/02/28/WSM-MO-006-GLAZED-CLI-MIGRATION--port-all-wsm-cli-verbs-to-glazed-commands/tasks.md — Execution checklist by phase and command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/ttmp/2026/02/28/WSM-MO-006-GLAZED-CLI-MIGRATION--port-all-wsm-cli-verbs-to-glazed-commands/reference/01-investigation-diary.md — Detailed diary
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/root.go — Migrated root initialization and new group registration
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/common/runtime.go — Shared `output-mode` and row emission helpers
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/workspace/add.go — Migrated `add` Glazed command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/workspace/remove.go — Migrated `remove` Glazed command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/workspace/status.go — Migrated `status` Glazed command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/git/diff.go — Migrated `diff` Glazed command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/git/log.go — Migrated `log` Glazed command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/git/branch.go — Migrated `branch create/switch/list` Glazed subcommands
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/git/commit.go — Migrated `commit` Glazed command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/workspace/create.go — Migrated `create` Glazed command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/workspace/fork.go — Migrated `fork` Glazed command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/workspace/delete.go — Migrated `delete` Glazed command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/workspace/merge.go — Migrated `merge` Glazed command
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/cmds/git/rebase.go — Migrated `rebase*` Glazed commands
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds — Legacy command layer removed
