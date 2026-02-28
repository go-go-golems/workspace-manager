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
