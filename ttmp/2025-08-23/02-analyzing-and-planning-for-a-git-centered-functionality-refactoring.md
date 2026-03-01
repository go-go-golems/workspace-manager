### 1. Purpose and Scope

This document proposes a refactoring of git- and workspace-related operations in `workspace-manager` to rely primarily on a Go git library instead of shelling out to the `git` command, and to enable safe concurrency for multi-repository operations. The goals are to improve reliability, testability, and performance, while retaining the ability to fall back to CLI for features not yet supported (notably `git worktree`).

---

### 2. Current State Assessment

**Reasoning:**
- Git is currently invoked via `os/exec` throughout `pkg/wsm`.
- Key files and operations:
  - `pkg/wsm/workspace.go`:
    - `ExecuteWorktreeCommand()` for `git worktree add` and variants.
    - `CheckBranchExists()`, `CheckRemoteBranchExists()` using `git show-ref`.
    - Worktree creation/removal; metadata and setup script orchestration.
  - `pkg/wsm/git_operations.go` (multi-repo ops):
    - Status parsing via `git status --porcelain`.
    - Stage/unstage via `git add`/`git reset`.
    - Commit (`git commit -m`) and push (`git push`).
    - Diffs via `git diff`/`git diff --cached`.
  - `pkg/wsm/discovery.go` (repository discovery):
    - Remote URL, current branch, branches, tags, last commit via various `git` calls.
  - `pkg/wsm/status.go` (workspace status):
    - Branch, modified/staged/untracked files, ahead/behind, conflicts using `git`.
  - `pkg/wsm/git_utils.go`:
    - `CheckBranchMerged()` using `git merge-base --is-ancestor`.
    - `CheckBranchNeedsRebase()` using `git rev-list HEAD..origin/main`.
  - `pkg/wsm/sync_operations.go`:
    - `pull` (with optional `--rebase`), `push`, ahead/behind, conflicts, branch create/switch, and logs via CLI.
- Interactive prompts in `workspace.go` and add/remove flows block concurrency and automation.

**Conclusion:**
- The system uses the CLI for almost all git behavior, including `worktree`, status, stage/unstage, commit, push, fetch, ahead/behind, and logs.
- There is a clear opportunity to centralize git interactions behind an interface and replace most with a Go library; `worktree` needs a special plan.

---

### 3. Library Evaluation: go-git vs git2go vs Hybrid

**Reasoning:**
- Options:
  - `go-git` (`github.com/go-git/go-git/v5`): Pure Go, no CGO, widely used. Supports repo open/clone, status, add/reset, commit, push/fetch, checkout/branch operations. Does not provide native Git "worktree" (multi-working-directory) management; `Worktree` in go-git is the in-repo working tree abstraction, not the `git worktree` feature.
  - `git2go` (libgit2 bindings): Offers broader coverage of Git features including `worktree` APIs (`git_worktree_*`). Requires CGO and a system `libgit2`, complicating installation and cross-platform builds.
  - Hybrid: Use `go-git` for the majority (status, add, commit, push, diff, branches, logs), and keep CLI or `git2go` for `worktree` and rebase-like features. This reduces complexity while capturing most benefits.
- Constraints from current codebase:
  - `git worktree` is central to workspace creation, add/remove, and deletion. Replacing it entirely would require either `git2go` or emulating shared-object multiple checkouts (non-trivial and risks data integrity).
  - Status and commit flows are prime candidates for go-git.

**Conclusion:**
- Adopt a hybrid approach:
  - Primary backend: `go-git` for repository operations (status, stage/unstage, commit, push, fetch, branch, diff, log, ahead/behind via graph traversal).
  - `worktree` management: continue with CLI in the first phases; optionally introduce `git2go` as an alternative backend later for environments that can support CGO.
  - Implement a backend-agnostic `GitClient`/`WorktreeManager` interface layer to enable future swaps.

---

### 4. Target Architecture and Interfaces

**Reasoning:**
- Introduce a new package (proposed: `pkg/wsm/gitclient`) that encapsulates all git calls via interfaces with context and structured error wrapping. Provide multiple implementations and a hybrid that delegates by capability.
- Keep worktree management in a separate interface to avoid coupling to `go-git` limitations.
- Keep the CLI layer (`cmd/cmds`) unchanged; migrate business logic in `pkg/wsm` to depend only on interfaces.

**Proposed structure:**
- `pkg/wsm/gitclient/`:
  - `client.go` (interfaces and factories)
  - `gogit_client.go` (go-git based implementation)
  - `cli_client.go` (current CLI-based implementation, adapted)
  - `libgit2_client.go` (optional future)
  - `hybrid_client.go` (delegates; e.g., go-git for most, CLI for worktree/rebase)
  - `graph.go` (ahead/behind helpers for go-git)

**Interfaces (sketch):**
- `GitClient` for repo-level operations:
```go
// pkg/wsm/gitclient/client.go
package gitclient

type RepositoryHandle interface { Path() string }

// Optional: compile-time checks
// var _ GitClient = (*GoGitClient)(nil)
// var _ GitClient = (*CliGitClient)(nil)

type GitClient interface {
  Open(ctx context.Context, repoPath string) (RepositoryHandle, error)

  // Introspection
  CurrentBranch(ctx context.Context, repo RepositoryHandle) (string, error)
  RemoteURL(ctx context.Context, repo RepositoryHandle, remote string) (string, error)
  ListBranches(ctx context.Context, repo RepositoryHandle) ([]string, error)
  ListTags(ctx context.Context, repo RepositoryHandle) ([]string, error)
  LastCommit(ctx context.Context, repo RepositoryHandle) (string, error)

  // Working tree
  Status(ctx context.Context, repo RepositoryHandle) (Status, error)
  Add(ctx context.Context, repo RepositoryHandle, path string) error
  Reset(ctx context.Context, repo RepositoryHandle, path string) error
  Commit(ctx context.Context, repo RepositoryHandle, msg string, opts CommitOptions) (string, error)
  Diff(ctx context.Context, repo RepositoryHandle, staged bool) (string, error)

  // Sync
  Fetch(ctx context.Context, repo RepositoryHandle, remote string) error
  Push(ctx context.Context, repo RepositoryHandle, remote string) error
  AheadBehind(ctx context.Context, repo RepositoryHandle, upstream string) (ahead int, behind int, err error)

  // Branches
  CreateBranch(ctx context.Context, repo RepositoryHandle, name string, track bool, baseRef string) error
  CheckoutBranch(ctx context.Context, repo RepositoryHandle, name string, create bool, force bool) error
}

type WorktreeManager interface {
  Add(ctx context.Context, repoPath string, branch string, targetPath string, opts WorktreeAddOptions) error
  Remove(ctx context.Context, repoPath string, targetPath string, force bool) error
  List(ctx context.Context, repoPath string) ([]WorktreeInfo, error)
}
```

- Support types: `Status` (maps file status to staged/unstaged/untracked), `CommitOptions`, `WorktreeAddOptions`, `WorktreeInfo`.
- Design choices:
  - All methods accept `context.Context`.
  - `GitClient` is oblivious to how worktrees are implemented. `WorktreeManager` defaults to CLI; later, a libgit2-backed alternative can be introduced.
  - Use `github.com/pkg/errors` for wrapping.
  - Provide compile-time interface checks: `var _ GitClient = (*GoGitClient)(nil)`.

**Conclusion:**
- Introduce backend abstractions that allow a gradual, low-risk migration and enable concurrent execution with consistent cancellation and logging.

---

### 5. Concurrency Model

**Reasoning:**
- Many operations iterate per repository (status, add/remove, sync, commit, push). These are largely independent and can run concurrently.
- Constraints:
  - Limit parallelism (IO and network bound): use a semaphore or worker pool.
  - Preserve readable output: aggregate results per repo; avoid interleaved `stdout` pollution; use structured logs.
  - Avoid interactive prompts in concurrent paths; provide policy-driven decisions up-front.

**Design:**
- Use `errgroup` with a bounded concurrency semaphore:
```go
// pkg/wsm/internal/concurrency.go
sem := semaphore.NewWeighted(int64(maxJobs))
var g errgroup.Group
for _, repo := range workspace.Repositories {
  repo := repo
  if err := sem.Acquire(ctx, 1); err != nil { return err }
  g.Go(func() error {
    defer sem.Release(1)
    return doOperation(ctx, repo)
  })
}
if err := g.Wait(); err != nil { return err }
```
- Provide a `--jobs` flag (default 4-8) for commands that can parallelize (`sync`, `status`, `commit --add-all`, `push`, `diff`, branch ops) and a config default.
- Output strategy: collect per-repo results (structs) and render after completion, or stream via a channel with per-repo sections.

**Conclusion:**
- Adopt `errgroup` plus a semaphore to safely parallelize per-repo operations, with cancellation via `context.Context` and predictable output.

---

### 6. Mapping Current Functions to New Interfaces

**Reasoning:**
- Replace direct CLI invocations with `GitClient` where feasible; keep `WorktreeManager` via CLI for now.
- High-value replacement targets:
  - `pkg/wsm/git_operations.go`:
    - `GetWorkspaceChanges` → `GitClient.Status` mapping to `Status` per repo.
    - `StageFile`/`UnstageFile` → `GitClient.Add`/`GitClient.Reset`.
    - `CommitChanges` → `GitClient.Commit` and `GitClient.Push`.
    - `GetDiff` → `GitClient.Diff(staged)`.
  - `pkg/wsm/status.go`:
    - Replace all branch/file-status/ahead-behind/conflicts with `GitClient` methods.
  - `pkg/wsm/discovery.go`:
    - `RemoteURL`, `CurrentBranch`, `ListBranches`, `ListTags`, `LastCommit` via `GitClient`.
  - `pkg/wsm/sync_operations.go`:
    - `pull` → `GitClient.Fetch` + fast-forward/merge strategy if/when supported; else keep CLI for pull/rebase temporarily.
    - `push` → `GitClient.Push`.
    - `ahead/behind` → `GitClient.AheadBehind` (implement in go-git with graph walks; fallback to CLI for initial phase).
  - `pkg/wsm/workspace.go`:
    - Worktree creation/removal/list → `WorktreeManager` (CLI for now).
    - Branch existence checks → `GitClient` branch APIs.

**Conclusion:**
- Most of `git_operations.go`, `status.go`, and large parts of `discovery.go` can migrate to `go-git` quickly. `worktree` flows remain CLI-backed pending `git2go` or a stable alternative.

---

### 7. Non-Interactive Policies and UX

**Reasoning:**
- Interactive prompts block concurrency and automation.
- Decisions to capture:
  - When branch exists: overwrite/use/cancel.
  - Forced removal with untracked files.

**Design:**
- Introduce flags and config:
  - `--non-interactive` global flag; `--yes` accept overwrite/remove; `--policy=<overwrite|use|cancel>` for branch-exists.
  - For removals: `--force-worktrees` already exists; add `--assume-yes` to avoid prompts.
- Place all prompting in CLI layer, not in `pkg/wsm`.

**Conclusion:**
- Decouple prompts from core logic; core receives an explicit policy enum. This unblocks parallel execution and headless usage.

---

### 8. Migration Plan (Phased)

**Reasoning:**
- Reduce risk via incremental swaps and a runtime-selectable backend.

**Phases:**
- Phase 0: Introduce interfaces and wire hybrid backend selection
  - Add `pkg/wsm/gitclient` with `GitClient`/`WorktreeManager` and a `HybridClient` that uses go-git for supported calls and CLI for the rest.
  - Config: `WSM_GIT_BACKEND=hybrid|cli|gogit|libgit2` and a `--git-backend` flag.
- Phase 1: Port discovery/status/diff
  - Replace `discovery.go` and `status.go` with go-git; keep ahead/behind via CLI initially if needed.
  - Port `GetDiff` to go-git.
- Phase 2: Port stage/commit/push
  - Replace `StageFile`, `UnstageFile`, `CommitChanges`, and `Push`.
- Phase 3: Implement ahead/behind in go-git
  - Implement a commit-graph walk to compute ahead/behind against `@{upstream}` equivalent.
  - Add tests comparing results to CLI for confidence.
- Phase 4: Pull/Rebase
  - Pull fast-forward can be emulated; rebase remains complex—keep CLI or consider `git2go` if needed.
- Phase 5: Worktree
  - Continue CLI; optionally implement `libgit2` backend for environments that need fully native worktree ops.
- Phase 6: Enable concurrency with `--jobs`
  - Parallelize operations guarded by `errgroup` and a semaphore; ensure `--non-interactive` policies are in place.

**Conclusion:**
- A staged approach keeps the system stable while increasing native Go coverage and unlocking concurrency.

---

### 9. Testing Strategy

**Reasoning:**
- We need confidence that go-git behavior matches or is acceptable vs CLI, especially for status and ahead/behind.

**Plan:**
- Unit tests for `gitclient/gogit_client.go` using temp repos:
  - Create repos with commits/branches/tags; verify Status/Add/Commit/Push to a local bare remote.
  - Compare `AheadBehind` against CLI-calculated values for fixtures.
- Integration tests for workspace flows:
  - Create a synthetic workspace with 2–3 repos; run `status`, `commit`, `push`, `diff`, `sync` (fetch-only initially).
- Concurrency tests:
  - Run multi-repo operations with `--jobs>1`, assert no data races and consistent results; use `-race` in CI.
- Backends parity tests:
  - Execute the same test suite with `WSM_GIT_BACKEND=cli` and `gogit` to compare outputs.

**Conclusion:**
- Comprehensive test coverage around the new client ensures smooth migration and maintainability.

---

### 10. Risks and Mitigations

**Reasoning:**
- `go-git` limitations:
  - No native `git worktree` support; rebase semantics limited; ahead/behind requires custom graph logic.
- `git2go` complexity:
  - CGO and `libgit2` installation friction; cross-platform hurdles.
- Behavioral diffs:
  - Porcelain vs go-git `Status` representations may differ (rename detection, etc.).

**Mitigations:**
- Maintain a hybrid backend with CLI fallback.
- Keep `worktree` and complex sync (rebase) on CLI initially.
- Golden tests comparing outputs across backends.
- Feature flags and per-command `--git-backend` override.
- See `ttmp/2025-08-23/03-status-and-design-of-wsm-rebase-and-conflict-management.md` for the rebase/conflict handling plan and TODOs.

**Conclusion:**
- The hybrid strategy de-risks the transition while delivering immediate benefits.

---

### 11. Implementation Checklist (High-Level)

- [x] Add `pkg/wsm/gitclient` with `GitClient`, `WorktreeManager`, DTOs.
- [x] Implement `GoGitClient` (status, add/reset, commit, diff, push, fetch, branches, tags; returns `ErrNotImplemented` where needed).
- [x] Implement `CliGitClient` (wrap existing exec logic in one place).
- [x] Implement `HybridClient` composition.
- [x] Refactor `pkg/wsm/git_operations.go` to use `GitClient`.
- [x] Refactor `pkg/wsm/status.go` and `pkg/wsm/discovery.go` to use `GitClient`.
- [ ] Refactor `pkg/wsm/sync_operations.go` completely to `GitClient` (partial: fetch/push/aheadBehind/branch done; rebase + conflict harmonization pending).
- [x] Refactor `pkg/wsm/workspace.go` to call `WorktreeManager` and remove inline prompts; move policies to CLI flags. (Initial pass complete; cleanup of remaining exec verification listings pending.)
- [x] Add concurrency helpers and `--jobs` flags to eligible commands.
- [ ] Add tests (unit+integration+concurrency) and run against CLI/gogit backends.
- [ ] Document `WSM_GIT_BACKEND` and new flags in `README.md`.

### 17. Remaining Verbs and Status

- **sync**: [in-progress] now uses `GitClient` for fetch/push/ahead/behind and branch ops; rebase path is not implemented; conflict detection is still CLI-based.
- **branch**: [migrated] create/switch paths use `GitClient`.
- **diff**: [migrated] via `GitClient.Diff` with hybrid fallback (falls back to CLI for unified diff where needed).
- **commit**: [migrated] staging/commit/push via `GitClient`.
- **status**: [migrated] via `GitClient.Status` and `AheadBehind`.
- **discover**: [migrated] via `GitClient` with CLI fallback when needed.
- **worktree (create/add/remove/list)**: [migrated] implemented via `WorktreeManager` (CLI-backed); some verification/listing still uses exec; follow-up cleanup pending.

### 18. Why multiple backends (not just one)?

- **Coverage gaps**: `go-git` lacks native `git worktree` and certain porcelain-equivalent behaviors; `git2go` requires CGO and libgit2 installation. The hybrid approach delivers value immediately while retaining features.
- **Portability**: Pure Go (`go-git`) keeps builds simple and portable by default. CLI fallback ensures edge cases and complex operations still work everywhere `git` is available.
- **Risk management**: Gradual migration allows parity checks and rollback. Once coverage and confidence improve, we can collapse to a single backend per deployment environment.
- **Future optionality**: Some users may prefer pure-Go (no `git` dependency); others may prefer system `git` parity; some environments can support `git2go` for full native APIs. The abstraction keeps that door open.

### 19. New TODOs

- [x] Implement `--jobs` concurrency (status, diff, commit, push, sync fetch) with `errgroup` + semaphore.
- [ ] Add non-interactive policy flags and remove prompts from `workspace.go` (centralize in CLI).
- [ ] Evaluate adding a `git2go` backend for `worktree` and rebase (document CGO trade-offs).
- [ ] Write unit tests for `gogit_client` and backends parity tests; add integration tests for key verbs.

### 20. Work Log (2025-08-23)

- Created `pkg/wsm/gitclient` abstraction layer:
  - `client.go` (interfaces and types), `cli_client.go` (CLI-based), `gogit_client.go` (go-git-based), `hybrid_client.go` (fallback composition), `worktree_cli.go` (`git worktree` operations via CLI).
  - Added `pkg/wsm/git_integration.go` with `BuildGitBackends()` selector (env `WSM_GIT_BACKEND`).
- Refactored modules to use new clients:
  - `status.go`: switched to `GitClient.Status`/`AheadBehind`; removed legacy exec helpers.
  - `git_operations.go`: switched to `GitClient` (status, add/reset, commit, push, diff).
  - `discovery.go`: prefer `GitClient` for repo metadata with CLI fallbacks.
  - `sync_operations.go`: switched fetch/push/ahead-behind and branch create/switch to `GitClient`; kept rebase and conflict detection as-is (CLI) for now.
  - `workspace.go`: switched worktree add/remove to `WorktreeManager`; removed interactive prompts; retained some exec-based verification listings to be cleaned.
- Verified safe, read-only commands:
  - `wsm --help`, `list workspaces`, `list repos`, `status`, `diff`, `sync all --dry-run` run successfully with `WSM_GIT_BACKEND=hybrid`.
- Fixed compile issues during migration (unused imports, symbol qualification, small API differences in go-git).

### 21. Lessons Learned and Gotchas

- `go-git` API differences:
  - Path-specific mixed resets aren’t supported; use hybrid fallback (CLI) or redesign unstaging flows.
  - Unified diffs are non-trivial to generate with `go-git`; hybrid fallback is pragmatic.
  - `Worktree` in go-git is not `git worktree`; keep using CLI for multi-working-directory semantics until we add a native backend.
- Keep prompts out of core logic: they block concurrency and complicate testing. Push policy decisions (overwrite/use/cancel, force removal) to CLI flags/options.
- Parity checks are essential: compare outputs (status, ahead/behind) under CLI vs go-git during migration.
- Logging and UX: continue to separate user-facing prints (`pkg/output`) from structured logging; avoid interleaving stdout from concurrent operations.
- Concurrency defaults: a conservative default of `--jobs=1` avoids surprises; users can scale up as needed. Ordered summaries remain readable when results are collected per-repo before printing.
- Worktree verification: using `WorktreeManager.List()` eliminates brittle parsing of `git worktree list` output and keeps verification backend-agnostic.

### 22. Expanded Next Steps (for the incoming developer)

- Refine sync and workspace flows:
  - [ ] Implement or explicitly document rebase handling strategy. Options: keep CLI only; or add `git2go` backend; or instruct users to rebase manually.
  - [ ] Unify conflict detection under `GitClient` status model (or keep CLI with clear rationale).
  - [x] Replace remaining exec-based verification/listing (e.g., worktree list) with `WorktreeManager.List()` and tidy any leftover exec code.
- Concurrency and flags:
  - [x] Add `--jobs` to relevant verbs (`status`, `diff`, `commit` with `--add-all`, `push`, `sync`).
  - [x] Implement `errgroup` + semaphore with sensible default (e.g., 6); ensure ordered, readable summaries. (Initial default wired as `--jobs`, core can choose default later.)
  - [ ] Add `--non-interactive`/`--assume-yes` and policy flags to CLI where needed; plumb into core as enum/policy instead of prompting.
- Rebase and conflicts (see `ttmp/2025-08-23/03-status-and-design-of-wsm-rebase-and-conflict-management.md`):
  - [x] CLI-backed rebase helpers (`start|continue|abort|status`) and `sync --rebase` wiring.
  - [x] `rebase` command: `--jobs`, `--manual`, and `status|continue|abort` subcommands.
  - [x] `conflicts` command: `list|open|mark-resolved` helpers.
  - [ ] Optional `tui conflicts` and tests/documentation follow-up.
- Testing and documentation:
  - [ ] Add unit tests for `gogit_client` (status, add, commit, push, ahead/behind) using temp repos + local bare remotes.
  - [ ] Add integration tests across a small temp workspace (2–3 repos); compare outputs between backends (CLI vs go-git) where feasible.
  - [ ] Document `WSM_GIT_BACKEND` and how to run safe read-only checks in `README.md`.
- Deprecation plan:
  - [ ] Decide default backend (`hybrid` now). After parity + tests: consider `gogit` default for most operations while keeping `WorktreeManager` CLI.
  - [ ] Once stable: remove unused backends or leave as opt-in (env/flag) per deployment policy.

### 23. Quick Reference (for daily use)

- Build and safe checks:
  - `go build ./cmd/wsm`
  - `WSM_GIT_BACKEND=hybrid ./demo/wsm list workspaces`
  - `WSM_GIT_BACKEND=hybrid ./demo/wsm status`
  - `WSM_GIT_BACKEND=hybrid ./demo/wsm diff`
  - `WSM_GIT_BACKEND=hybrid ./demo/wsm sync all --dry-run`
- Where to change backends: env `WSM_GIT_BACKEND` (`hybrid` | `gogit` | `cli`).
- Key code entry points:
  - Clients: `pkg/wsm/gitclient/*.go`; selection: `pkg/wsm/git_integration.go`.
  - Core flows: `pkg/wsm/status.go`, `pkg/wsm/git_operations.go`, `pkg/wsm/sync_operations.go`, `pkg/wsm/workspace.go`.

### 24. Backend Deprecation Plan

- Short term: keep `hybrid` default for maximum coverage; retain CLI for `worktree` and rebase.
- Mid term: after tests + parity are green, set default to `gogit` for repo operations; keep `WorktreeManager` CLI.
- Long term: if a native `worktree` backend is adopted (e.g., `git2go`), consider removing CLI fallback; otherwise keep CLI as optional fallback behind a flag.


### 25. Today’s updates (2025-08-23)

- Implemented concurrency with `errgroup` + `semaphore` in `status` and `sync` flows; added `--jobs` flags in CLI.
- Replaced `git worktree list` verification with `WorktreeManager.List()` in `workspace.go` cleanup paths.
- Ensured `golang.org/x/sync` is a direct dependency.
- Notes:
  - Default `--jobs` is conservative (1); users can scale up as needed.
  - Parallel execution collects results per repo to keep output readable and ordered.
  - Next to consider: non-interactive/policy flags, and porting concurrency to `diff`, `commit --add-all`, and other eligible verbs if desired.
