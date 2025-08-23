### 1. Purpose

This document captures the current status, complexities, and proposed design for rebase and conflict handling in `workspace-manager` (WSM). It aims to clarify what we support today, what “good” looks like, and how to evolve the system with minimal risk.

---

### 2. Current State (2025-08-23)

- WSM uses a hybrid Git backend: a Go client (`go-git`) for most repo-level operations and CLI fallbacks, with `git worktree` still CLI-backed.
- Rebase is not implemented in the Go client path. `sync` currently performs `Fetch` and `Push` via the client; the `--rebase` flag is accepted but logs a warning and does not rebase.
- Conflict detection in `sync` is CLI-based: we parse `git status --porcelain` for conflict codes (UU/AA/DD, etc.).
- Status and ahead/behind information are provided via `GitClient` for most cases; ahead/behind uses client computation when available.
- Concurrency is available for multi-repo `status`, `diff`, and `sync` with `--jobs` using `errgroup` + `semaphore`.

Implications:
- Users must handle rebase manually or outside WSM if they require it.
- Conflicts are detected post-operation in a best-effort manner; there is no guided resolution flow.

---

### 3. What Rebase/Conflict Handling Entails

- Rebase semantics:
  - Prepare: fetch/update remote references; determine upstream (`@{upstream}` or configured remote/branch).
  - Operation: replay local commits on top of the upstream tip; manage stops at conflicts; allow continue/skip/abort.
  - Safety: preserve work, ensure clean working tree (or explicit policy to allow dirty states), stash handling.
  - UX: clear progress reporting; resumability across repos; consistent policy defaults.

- Conflict handling:
  - Detection: identify conflicted paths, surface details (both-sides edit, add/add, delete/modify, etc.).
  - Assistance: offer commands to open editors/mergetool, stage resolutions, continue/abort; batch summary across repos.
  - Persistence: allow resuming a multi-repo rebase (track per-repo state: ongoing, paused, conflicted, resolved).

- Multi-repo coordination:
  - Per-repo independence with bounded concurrency; avoid deadlocks by not interleaving interactive prompts.
  - Ordered, readable summary output after operations; channel streaming optional but avoid stdout mixing.
  - Policies must be explicit (non-interactive flags) to run parallel operations safely.

---

### 4. Complexities and Trade-offs

- Library limitations:
  - `go-git` lacks native rebase; implementing rebase requires custom logic or CLI/libgit2.
  - `git worktree` is CLI-only in our system; rebase within worktrees must respect that environment.

- Behavioral edge cases:
  - Diverse upstream configurations (tracking branches missing, detached HEAD, diverged histories).
  - Dirty working trees (untracked/staged changes) and stashing strategy.
  - Binary files, submodules, large histories, rename detection policies.

- Multi-repo orchestration:
  - Fail-fast vs best-effort: whether one repo failure halts others.
  - Rollback/abort across repos is not transactional.
  - Output consistency vs real-time feedback when running concurrently.

- Portability vs parity:
  - CLI provides battle-tested behavior and parity with user expectations.
  - libgit2 (`git2go`) adds coverage but increases build complexity (CGO, system libs).

---

### 5. Design Options

1) CLI-first rebase (short-term pragmatic)
- Keep rebase exclusively via CLI (`git rebase`), orchestrated by WSM per-repo.
- Model: WSM detects upstream, runs `git fetch`, then `git rebase --rebase-merges?` as configured.
- Conflict detection: use `git status --porcelain` + `rebase --show-current-patch` context when paused; summarize per repo.
- Continue/abort: add WSM verbs `rebase continue|abort` that iterate repos with ongoing rebases.
- Pros: High parity and low effort; no CGO.
- Cons: Shelling out; less testable; platform assumptions.

2) libgit2-backed rebase (mid-term, optional)
- Add a `libgit2` backend implementing `Rebase(...)` and richer conflict info, gated by `WSM_GIT_BACKEND=libgit2`.
- Keep hybrid fallback to CLI if libs not present.
- Pros: Native control, testability, no external git dependency.
- Cons: CGO complexity; cross-platform packaging.

3) Guidance-only mode (documentation-first)
- Do not implement rebase; provide a guided flow that shows per-repo steps to run manually with copy/paste commands.
- Pros: Zero risk; simplest.
- Cons: No automation.

Recommendation:
- Phase A (now): Implement CLI-first rebase with non-interactive policies, plus detection and reporting.
- Phase B (optional): Add `libgit2` backend; keep hybrid selection.

---

### 6. Proposed API and UX Changes

- Core API additions (GitClient facade stays unchanged for now):
  - New `SyncOptions.Rebase` behavior: if `Rebase=true`, perform CLI rebase per repo (Phase A).
  - Add `RebaseOperations` helper (Phase A) encapsulating CLI calls: `RebaseStart`, `RebaseContinue`, `RebaseAbort`, `RebaseStatus`.
  - Add `ConflictInfo` struct for richer reporting (files, types, hints). Initially populated via porcelain parsing.

- Modes:
  - Automated mode (default): WSM orchestrates per-repo rebase and conflict detection non-interactively, failing fast and summarizing results.
  - Manual mode (opt-in): user handles rebase/conflict resolution directly with their tools; WSM provides discovery, summaries, and helper commands but does not attempt automated rebase. Toggle via `--manual` or config.

- CLI commands:
  - `wsm sync pull --rebase [--non-interactive] [--jobs N]` (runs `git fetch && git rebase` per repo; non-interactive mode fails fast on conflicts with summary).
  - `wsm rebase continue|abort [--jobs N]` (resume/abort ongoing rebases across repos).
  - `wsm rebase status` (report per-repo rebase state + conflicts summary).
  - Manual mode switch: `--manual` can be used on commands like `sync pull --rebase` to skip automation and only provide summaries and helpers.

- Standard helpers for common situations:
  - Conflicts overview: `wsm conflicts list [--repo <name>]` (shows files and conflict types).
  - Open resolver: `wsm conflicts open [--repo <name>] [--tool <mergetool>]` (launch editor/mergetool on conflicted files).
  - Mark resolved: `wsm conflicts mark-resolved [--repo <name>] [--all]` (stage resolved files).
  - Rebase controls: `wsm rebase continue|skip|abort [--repo <name>]`.

- TUI helpers (optional):
  - `wsm tui conflicts` launches an interactive view for selecting conflicts, opening files, staging resolutions, and invoking `continue/abort`.
  - TUIs should operate per-repo under the hood and respect `--repo` filters and `--jobs` limits.

- Policies and flags:
  - `--manual` (disable automated rebase and resolution; provide summaries and helper commands only).
  - `--non-interactive` (no prompts; exit with actionable summary on conflicts).
  - `--assume-yes` for operations that can be destructive (not directly for rebase, but for related flows).
  - Configurable upstream resolution: default to `@{upstream}`, fallback to `origin/<current-branch>`.

---

### 7. Minimal Implementation Plan (Phase A, CLI-first)

- Add `pkg/wsm/rebase_operations.go`:
  - `DetectUpstream(ctx, repoPath) (string, error)`
  - `Start(ctx, repoPath, upstream string, opts RebaseOptions) error` (shells out to `git rebase`)
  - `Continue(ctx, repoPath) error`, `Abort(ctx, repoPath) error`
  - `Status(ctx, repoPath) (RebaseState, []ConflictInfo, error)`
- Integrate into `SyncOperations.pullRepository` when `Rebase=true`:
  - Replace fetch-only with `fetch && rebase` (non-interactive by default); populate `SyncResult.Conflicts` and error message if stopped.
- Add CLI:
  - `wsm rebase status|continue|abort` (iterate repos; `--repo` filter; `--jobs`).
- Output:
  - Summarize conflicts per repo with counts and representative paths.

---

### 8. Data Structures

- `type RebaseState string` → values: `none`, `in-progress`, `stopped-conflicts`, `completed`.
- `type ConflictInfo struct { File string; Type string; Hints []string }`
- `type RebaseOptions struct { PreserveMerges bool; Autosquash bool }`

---

### 9. Testing Strategy

- Unit tests against synthetic repos:
  - Linear rebase with no conflicts.
  - Conflicting rebase that stops with UU files.
  - Aborted rebase rolls back correctly; status detection reports `none` afterwards.
- Integration tests:
  - Multi-repo workspace with mixed outcomes; ensure reporting is stable and concurrency safe (`--jobs > 1`).
- Backend parity tests (future):
  - If `libgit2` backend is added, compare outputs vs CLI for small fixtures.

---

### 10. Risks and Mitigations

- Risk: interactive prompts during rebase break concurrency.
  - Mitigation: enforce `--non-interactive` and fail fast; provide `continue/abort` follow-ups.
- Risk: divergent upstreams or missing tracking.
  - Mitigation: detect/upfront check; provide clear error and suggested upstream.
- Risk: partial success across repos.
  - Mitigation: summarize; never hide failures; keep operations idempotent.

---

### 11. Next Steps

- Implement Phase A (CLI-first): rebase start/continue/abort/status helpers and wire into `sync pull --rebase`.
- Add `--manual` mode to allow users to handle rebase/conflicts manually while WSM provides summaries and helpers.
- Add `--non-interactive` policies to relevant commands and surface explicit upstream.
- Introduce standard helpers (`conflicts list|open|mark-resolved`, `rebase continue|skip|abort`) and an optional `tui conflicts` view.
- Document user flows and troubleshooting in README.
- Evaluate interest/need for a `libgit2` backend (Phase B) after Phase A stabilizes.

### 12. Implementation TODOs (Phase A)

- [x] Add `pkg/wsm/rebase_operations.go` encapsulating CLI helpers (detect upstream, start, continue, abort, status).
- [x] Wire `sync pull --rebase` to use `RebaseOperations` (fetch + rebase, non-interactive; set `SyncResult.Conflicts`).
- [x] Enhance `rebase` command:
  - [x] Add `--jobs` for parallel per-repo execution.
  - [x] Add `--manual` to print suggested commands and skip automation.
  - [x] Add subcommands: `rebase status`, `rebase continue`, `rebase abort` (support `--repo`, `--jobs`).
- [x] Add standard helpers (initial CLI):
  - [x] `conflicts list [--repo]` (UU/AA/DD parsing and summary).
  - [x] `conflicts open [--repo] [--tool]` (launch editor/mergetool).
  - [x] `conflicts mark-resolved [--repo] [--all]` (stage resolved files).
- [ ] Optional: scaffold `tui conflicts` (list/select, open, stage, continue/abort).
- [ ] Documentation: update README with manual vs automated modes, flows, and troubleshooting.
- [ ] Tests: unit + integration for start/continue/abort/status across small fixture repos.
