# Tasks

## Phase 0: Analysis Baseline (Completed)

- [x] Inventory all root verbs and subcommands from `cmd/wsm/root.go` and `cmd/cmds/*`
- [x] Collect command complexity signals (LoC, shell-outs, concurrency, interaction)
- [x] Produce architecture analysis/design baseline for intern onboarding

## Phase 1: Plan Update for 6-Command Removal Sprint

- [x] Update implementation plan to remove these commands: `sync`, `conflicts`, `tmux`, `starship`, `pr`, `push`
- [x] Define consolidation mapping from removed verbs to retained workflows
- [x] Define compatibility policy: no backwards compatibility, hard removal
- [x] Define phased execution and commit boundaries

## Phase 2: CLI Surface Reduction (Code)

- [x] Remove root command registrations for the six removed commands from `cmd/wsm/root.go`
- [x] Delete command files:
- [x] `cmd/cmds/cmd_sync.go`
- [x] `cmd/cmds/cmd_conflicts.go`
- [x] `cmd/cmds/cmd_tmux.go`
- [x] `cmd/cmds/cmd_starship.go`
- [x] `cmd/cmds/cmd_pr.go`
- [x] `cmd/cmds/cmd_push.go`
- [x] Ensure no remaining references to removed command constructors

## Phase 3: Consolidation and Messaging Cleanup

- [x] Update root command long help text to remove sync/tmux/GitHub-specific claims
- [x] Update README command matrix and examples to remove removed verbs
- [x] Update IMPLEMENTATION guide command documentation to remove removed verbs
- [x] Add migration notes:
- [x] `sync` -> use `status`, `rebase`, `branch`, `commit --push`, direct git as needed
- [x] `conflicts` -> use `rebase status/continue/abort` plus direct `git mergetool` / `git add`
- [x] `pr`/`push` -> use native `gh`/`git` directly outside WSM
- [x] `tmux`/`starship` -> out of WSM scope

## Phase 4: Build/Test Validation

- [x] Run `go test ./cmd/... ./pkg/...` and capture failures
- [x] Run `go test ./...` and capture full-project baseline
- [x] Verify `wsm --help` no longer lists removed commands

## Phase 5: Diary, Commits, and Publishing

- [x] Update detailed implementation diary after each completed phase
- [x] Commit planning updates
- [x] Commit command removals
- [x] Commit docs/migration cleanup + validation updates
- [x] Run `docmgr doctor --ticket WSM-MO-005-CMD-PKG-REARCH-REMOVE-SYNC --stale-after 30`
- [x] Upload diary to reMarkable under today’s ticket folder
- [x] Update changelog and close completed tasks

## Phase 6: pkg-First Architecture Consolidation

- [x] Add `pkg/wsm/workspace_context.go` as the single workspace detection/loading service
- [x] Remove command-local workspace detection/loading implementations from `cmd_status.go` and `cmd_commit.go`
- [x] Add a thin shared command helper that delegates workspace resolution to `pkg/wsm/workspace_context.go`
- [x] Extract branch operations from `pkg/wsm/sync_operations.go` into `pkg/wsm/branch_operations.go`
- [x] Extract workspace log/history operations from `pkg/wsm/sync_operations.go` into `pkg/wsm/history_operations.go`
- [x] Update `cmd_branch.go` to use `BranchOperations` and branch-specific result DTOs
- [x] Update `cmd_diff.go` (`log` path) to use `HistoryOperations`
- [x] Remove stale `SyncOperations` branch/log code and sync-only leftovers from `pkg/wsm/sync_operations.go`
- [x] Migrate/replace `sync_operations_branch_test.go` with branch-operations tests
- [x] Run targeted tests for `cmd/cmds`, `pkg/wsm`, and updated command help

## Phase 7: Rebase/Merge Command Workflow Extraction

- [x] Introduce `pkg/wsm/workflows/rebase_workflow.go` for status/action fan-out now in `cmd_rebase.go`
- [x] Introduce `pkg/wsm/workflows/merge_workflow.go` for merge candidate planning/execution now in `cmd_merge.go`
- [x] Refactor `cmd_rebase.go` to thin adapter over workflow service
- [x] Refactor `cmd_merge.go` to thin adapter over workflow service
- [x] Add unit tests for new workflow services
- [x] Run `go test ./cmd/... ./pkg/...` and capture full results

## Phase 8: Ticket Wrap-up for pkg Consolidation

- [x] Update diary with per-task implementation notes and rationale
- [x] Update changelog with architecture consolidation summary
- [x] Run `docmgr doctor --ticket WSM-MO-005-CMD-PKG-REARCH-REMOVE-SYNC --stale-after 30`
- [x] Commit consolidation changes in logical intervals
