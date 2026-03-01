# Tasks

## Done

- [x] Create ticket workspace and baseline docs
- [x] Collect line-anchored evidence from workspace/gitclient code
- [x] Create and run reproducible experiment script
- [x] Write detailed bug report and fix design for intern handoff
- [x] Add `RemoteBranchExists` method to `GitClient` interface
- [x] Implement `RemoteBranchExists` in `CliGitClient`
- [x] Implement `RemoteBranchExists` in `GoGitClient`
- [x] Implement hybrid fallback behavior for `RemoteBranchExists`
- [x] Update `WorkspaceManager.CheckRemoteBranchExists` to use new API
- [x] Remove ignored errors in worktree branch-resolution path
- [x] De-duplicate duplicated branch-resolution logic into shared helper
- [x] Add backend unit tests for remote branch existence semantics
- [x] Add/extend workflow tests for create/add worktree branch selection inputs (`resolveBranchState` + backend matrix)
- [x] Re-run repro script and record post-fix output

## Validation Notes

- Targeted tests passed:
  - `go test ./pkg/wsm/gitclient -run 'Hybrid|RemoteBranch' -v`
  - `go test ./pkg/wsm -run 'CheckRemoteBranchExists|ResolveBranchState' -v`
- Broader run executed:
  - `go test ./...` still fails in integration scenarios due existing harness/stale-binary issues outside this ticket scope.
