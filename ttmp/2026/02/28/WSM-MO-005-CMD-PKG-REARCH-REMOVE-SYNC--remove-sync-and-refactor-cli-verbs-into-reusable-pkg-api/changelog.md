# Changelog

## 2026-02-28

- Initial workspace created


## 2026-02-28

Added exhaustive command-verb complexity audit and pkg extraction design document; added phased task plan to remove sync and move business logic from cmd layer into reusable pkg services; updated investigation diary with detailed step-by-step findings.

### Related Files

- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/ttmp/2026/02/28/WSM-MO-005-CMD-PKG-REARCH-REMOVE-SYNC--remove-sync-and-refactor-cli-verbs-into-reusable-pkg-api/design-doc/01-command-verb-complexity-audit-and-pkg-extraction-plan.md — Primary architecture analysis and migration plan
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/ttmp/2026/02/28/WSM-MO-005-CMD-PKG-REARCH-REMOVE-SYNC--remove-sync-and-refactor-cli-verbs-into-reusable-pkg-api/reference/01-investigation-diary.md — Detailed investigation diary in standard what-worked format
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/ttmp/2026/02/28/WSM-MO-005-CMD-PKG-REARCH-REMOVE-SYNC--remove-sync-and-refactor-cli-verbs-into-reusable-pkg-api/tasks.md — Phased execution checklist including sync removal


## 2026-02-28

Implemented six-command CLI surface reduction (removed sync, conflicts, pr, push, tmux, starship), updated root command help/registration, and consolidated README/IMPLEMENTATION migration guidance. Added detailed implementation diary entries and validation evidence.

### Related Files

- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/IMPLEMENTATION.md — Documented removed command surface and migration guidance
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/README.md — Removed deleted command references and added migration mapping
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/root.go — Removed six command registrations and updated root help text
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/ttmp/2026/02/28/WSM-MO-005-CMD-PKG-REARCH-REMOVE-SYNC--remove-sync-and-refactor-cli-verbs-into-reusable-pkg-api/reference/01-investigation-diary.md — Detailed implementation diary and validation notes


## 2026-02-28

Completed pkg-first architecture consolidation: added shared workspace context service, extracted branch/history services, removed SyncOperations, and moved rebase/merge orchestration into pkg/wsm/workflows with thin command adapters.

### Related Files

- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_merge.go — Now thin adapter over workflow service
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_rebase.go — Now thin adapter over workflow service
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/branch_operations.go — Branch operations extracted from deprecated sync service
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/history_operations.go — Workspace log/history operations extracted into dedicated service
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workflows/merge_workflow.go — Merge orchestration moved from command layer
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workflows/rebase_workflow.go — Rebase orchestration moved from command layer
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workspace_context.go — New canonical workspace detection/loading service
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/ttmp/2026/02/28/WSM-MO-005-CMD-PKG-REARCH-REMOVE-SYNC--remove-sync-and-refactor-cli-verbs-into-reusable-pkg-api/reference/01-investigation-diary.md — Detailed implementation diary for phase 6 and 7

