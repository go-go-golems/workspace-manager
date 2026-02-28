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
