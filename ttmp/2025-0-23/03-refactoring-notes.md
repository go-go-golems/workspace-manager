### Refactoring Notes: Git Client Migration and Concurrency Prep

Date: 2025-08-23

#### What was done

- Introduced `pkg/wsm/gitclient` abstraction:
  - `client.go`: `GitClient`, `WorktreeManager`, and support types.
  - `cli_client.go`: CLI-backed `GitClient` (uses `git` exec for full coverage).
  - `worktree_cli.go`: CLI-backed `WorktreeManager` for `git worktree` operations.
  - `gogit_client.go`: Go-git backed client for most repo operations; returns `ErrNotImplemented` where not supported.
  - `hybrid_client.go`: Primary/secondary composition; falls back when `ErrNotImplemented`.
- Added `pkg/wsm/git_integration.go`:
  - `BuildGitBackends(ctx)` chooses backend (`WSM_GIT_BACKEND` env: `hybrid` default).
- Integrated discovery with client fallback:
  - `pkg/wsm/discovery.go`: prefer `GitClient` for `RemoteURL`, `CurrentBranch`, `Branches`, `Tags`, `LastCommit`; fallback to CLI helpers when unavailable.
- Refactored status to client-based path:
  - `pkg/wsm/status.go`: `GetWorkspaceStatus` uses `getRepositoryStatusWithClient()` and `GitClient.Status`/`AheadBehind`.
- Refactored multi-repo ops:
  - `pkg/wsm/git_operations.go`: switched `GetWorkspaceChanges`, `StageFile`, `UnstageFile`, `CommitChanges`, and `GetDiff` to use `GitClient` methods.
- Build + verification (read-only):
  - Built `wsm` and ran: `--help`, `list workspaces`, `list repos`, `status`, and `diff` using `WSM_GIT_BACKEND=hybrid`.

#### Next steps

- Replace remaining exec usage progressively:
  - `sync_operations.go`: pull/push/ahead-behind/branch ops → `GitClient` (keep rebase via CLI for now or plan `git2go`).
  - `workspace.go`: continue using `WorktreeManager` (CLI) for add/remove/list; add non-interactive policy flags.
- Remove deprecated exec helpers as coverage reaches parity:
  - `status.go`: delete legacy helpers (`getCurrentBranch`, `getModifiedFiles`, etc.) once not referenced.
  - `git_operations.go`: removed `getRepositoryChanges`, `hasStagedChanges`, and exec-based commit/push already; validate no stale refs remain.
- Add concurrency where safe:
  - Introduce `--jobs` and semaphore + `errgroup` for per-repo loops (status, diff, commit, push, sync fetch).
- Testing:
  - Unit tests for `gogit_client` against temp repos; parity checks vs CLI.
  - Integration tests for status/diff/commit flows with `WSM_GIT_BACKEND=cli|gogit|hybrid`.
- Optional backends:
  - Consider `git2go` for native `worktree` and rebase support (CGO trade-offs).

#### Decisions and rationale

- Hybrid approach: maximizes immediate gains (status/diff/commit/push) while retaining CLI for `worktree` and advanced sync.
- No backwards-compat layer kept in refactored code paths; new interfaces are the source of truth.
- Minimal shim in status to compute only what we can reliably infer from go-git; conflicts/merged/rebase kept false/CLI-only for now.

#### Caveats and learnings

- go-git has differences vs porcelain; we map to a simpler model:
  - `Status`: staged/modified/untracked; rename/copy detection left for future.
  - `Reset(path)` not available as in CLI; `GoGitClient.Reset` returns `ErrNotImplemented` to let hybrid fall back.
  - Unified diff generation is non-trivial with go-git; hybrid falls back to CLI in diff.
- Keep user-facing output consistent while swapping internals; leverage `pkg/output` for messages.
- Ensure `go.work` and module versions are compatible (`go 1.24.3`).

#### Checklist (ongoing)

- [x] Add `gitclient` and hybrid backend
- [x] Refactor status to `gitclient`
- [x] Refactor git operations to `gitclient`
- [x] Build and verify read-only commands
- [ ] Refactor `sync_operations.go`
- [ ] Wire `--jobs` + concurrency helpers
- [ ] Add tests and parity checks
- [ ] Remove leftover exec helpers in status once unused

#### Notes for next developer

- Backend selection: export `WSM_GIT_BACKEND=hybrid|gogit|cli` when testing.
- When adding new Git features, prefer adding to `gitclient` first; keep CLI fallback only if strictly necessary.
- For `worktree`, stick to `WorktreeManager` (CLI) until a non-CLI backend is chosen.
- Avoid interactive prompts in `pkg/wsm`; keep them in CLI layer and pass policies into functions.
