### 1. Purpose

This document designs a full Dagger-based pipeline to run `workspace-manager` (WSM) integration tests in a reproducible containerized environment, locally and in CI, without hand-rolling Docker scripts.

---

### 2. Why Dagger for WSM

- Single, portable pipeline definition in Go.
- Reproducible containerized environment (pin images, no host leakage).
- Built-in caching (Go mod + build cache) to keep iteration fast.
- Easy matrices (backends: `cli`, `gogit`, `hybrid`), variants (race, coverage, smoke vs full).
- Artifact export (binaries, logs, JUnit) to host or CI.

---

### 3. High-level Goals

- Build `wsm` and run integration tests inside a controlled container.
- Support backend matrix (`WSM_GIT_BACKEND`), with default `hybrid` and optional `cli`/`gogit` passes.
- Provide flags for: race detector, coverage, smoke vs full suite, and parallel jobs.
- Export logs, junit (optional), and binary artifacts as needed.
- Run the same pipeline locally (`go run ./ci/dagger`) and in CI.

---

### 4. Repo Layout (proposed additions)

- `ci/dagger/main.go` → entrypoint to run the pipeline.
- `ci/dagger/pipeline.go` → pipeline construction (optionally inline with main.go if small).
- `test/integration/` → integration tests (helpers/scenarios, see 04-integration-tests-design.md).
- `Makefile` → targets: `dagger`, `dagger-test`, `dagger-test-backends`.

---

### 5. Dagger Setup

- Dependency: `go get dagger.io/dagger@v0.11.3` (or latest stable used by your environment).
- Runtime: Dagger engine via Docker (installed on host/CI runner).
- Base image (pin digest where possible): `golang:1.24-bullseye` + `git`.

---

### 6. Container Environment

- Base container: `golang:1.24-bullseye`.
- Install: `git`, `bash`, `ca-certificates` (and optional mergetools for UI-free tests).
- Workdir: `/workspace`.
- Caches:
  - `/go/pkg/mod` → Go modules cache.
  - `/root/.cache/go-build` → Go build cache.
- Env:
  - `HOME=/tmp/wsm-home` (isolated), write minimal `~/.gitconfig` with test user/email.
  - `WSM_GIT_BACKEND` set per matrix run.

---

### 7. Pipeline Design (Go, Dagger SDK)

- Steps (per backend variant):
  1) Prepare base container + install deps + mount code + mount caches.
  2) `go mod download` (cached).
  3) Build `wsm` (optionally export artifact).
  4) Run integration tests `go test ./test/integration/...` with flags:
     - `-v -count=1`
     - optional `-race` (toggle)
     - optional `-run` for smoke subset.
  5) Export logs, optional coverage/junit (if using gotestsum or `-json` parsing).

- Matrices:
  - Iterate over `[]string{"hybrid","cli","gogit"}`; subset via CLI flags.

- Artifacts:
  - Test logs to `/workspace/.out/logs/<backend>.txt`.
  - Optional: `coverage.out`, `junit.xml` if using gotestsum.

---

### 8. Example Pipeline (Go)

```go
// See ci/dagger/main.go in-repo for the runnable implementation.
```

Notes:
- Dagger caches module and build data across runs.
- Artifacts land under host `./.out/` by default.
- Adjust `-run` filters and test package path once helpers/scenarios exist.

---

### 9. Local Usage

- Build and run:
  - `go run ./ci/dagger --backends=hybrid,cli,gogit --race`
  - `go run ./ci/dagger --backends=hybrid --smoke`
- Makefile targets:
```
dagger:
	go run ./ci/dagger --backends=hybrid

dagger-test:
	go run ./ci/dagger --backends=hybrid,cli,gogit --race
```

---

### 10. CI Integration (GitHub Actions example)

```yaml
name: integration-tests
on:
  push:
    branches: [ main ]
  pull_request:

jobs:
  dagger:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24.x'
      - name: Set up Docker (for Dagger engine)
        uses: docker/setup-buildx-action@v3
      - name: Run Dagger pipeline (hybrid smoke)
        run: go run ./ci/dagger --backends=hybrid --smoke
      - name: Upload artifacts
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: wsm-integration-out
          path: ./.out
```

For nightly or a separate job, run full backends with `--race` and coverage.

---

### 11. Enhancements

- JUnit XML: use `gotestsum --junitfile .out/junit-<be>.xml --` to produce CI-friendly reports.
- Coverage merge: merge per-backend coverage files; upload to coverage service.
- Pin base image by digest for reproducibility.
- Add a `smoke` package with very fast end-to-end checks for PR gating; nightly executes the full suite.
- Parameterize Go version via env/flag.

---

### 12. Security & Secrets

- No external secrets required for current tests.
- If future tests need tokens (e.g., GitHub), inject via Dagger secrets API rather than env files.

---

### 13. Risks & Mitigations

- Engine availability: Ensure Docker or compatible Dagger engine present on CI runner.
- Long runs: Use caching and smoke/full split; shard matrix if needed.
- Flakiness: keep hermetic repos and avoid network-dependent steps.

---

### 14. Implementation TODOs

- [x] Add Dagger dependency and `ci/dagger` program.
- [x] Implement base container (Go 1.24.x + git) and artifact export.
- [x] Add backend matrix, build, and test steps; export `.out/` artifacts.
- [x] Wire Makefile targets.
- [ ] Integrate with `test/integration` helpers and scenarios from 04-design.
- [ ] Optional: gotestsum for JUnit + coverage aggregation.
- [ ] Pin base image digest; consider OCI cache for faster CI.

---

### 15. Next Integration Tests to Build (Detailed Spec)

Below is a step-by-step plan for the first wave of integration tests. Each test section lists:
- Files to create
- Setup (fixtures)
- Execution (CLI invocations)
- Assertions
- Backend coverage

All tests live under `test/integration/` using the standard Go test framework.

#### 15.1 Helpers and Sandbox

- Files:
  - `test/integration/helpers/sandbox.go`
  - `test/integration/helpers/git.go`
  - `test/integration/helpers/wsm.go`

- Setup scaffolding (suggested APIs):
```go
// sandbox.go
func NewSandbox(t *testing.T) *Sandbox
func (s *Sandbox) Cleanup()
func (s *Sandbox) SetBackend(be string)
// Directories
func (s *Sandbox) TempDir(parts ...string) string

// git.go
func (s *Sandbox) InitBareRepo(name string) string
func (s *Sandbox) InitRepo(name string, remoteURL string) string
func (s *Sandbox) CommitFile(repoPath, filename, content, message string)
func (s *Sandbox) CreateBranch(repoPath, name, base string)
func (s *Sandbox) Checkout(repoPath, name string)
func (s *Sandbox) IntroduceConflict(repoPath string, filename string)

// wsm.go
type RunResult struct { Stdout, Stderr string; ExitCode int }
func (s *Sandbox) RunWSM(ctx context.Context, args ...string) RunResult
func (s *Sandbox) BuildWSM(ctx context.Context) string // optional, or use installed binary
```

- Environment:
  - Set `HOME` to a temp dir; write `~/.gitconfig` with user/email.
  - Export `WSM_GIT_BACKEND` per test.

#### 15.2 Status + Diff (Smoke)

- Files:
  - `test/integration/scenarios/status_diff_test.go`

- Setup:
  - Create bare remote `remote.git`.
  - Create `repo1`, `repo2` with remote origin.
  - Commit files to each.
  - Optionally create a simple workspace or run commands with `--workspace` detection.

- Execution:
  - `wsm status --workspace <ws>`
  - `wsm diff --workspace <ws>`

- Assertions:
  - Output contains workspace header and repo entries.
  - Diff output contains repository headers and expected file modifications.

- Backends: run for `hybrid` and `cli` in CI; optionally `gogit` in nightly.

#### 15.3 Commit + Push

- Files:
  - `test/integration/scenarios/commit_push_test.go`

- Setup:
  - Repos `repo1`, `repo2` with bare remote.
  - Modify files in both.

- Execution:
  - `wsm commit --add-all -m "test commit" --push`

- Assertions:
  - HEAD advanced; remote refs updated.
  - `wsm status` reports no pending changes.

- Backends: `hybrid`, `cli`.

#### 15.4 Sync (Pull/Push) and Ahead/Behind

- Files:
  - `test/integration/scenarios/sync_test.go`

- Setup:
  - Diverge local and remote by adding commits on each side for a repo.

- Execution:
  - `wsm sync all --dry-run` then `wsm sync pull` and `wsm sync push`.

- Assertions:
  - Before/after ahead/behind counts change as expected.
  - Dry-run produces stable summary.

- Backends: `hybrid`.

#### 15.5 Worktree Create/Delete

- Files:
  - `test/integration/scenarios/worktree_test.go`

- Setup:
  - Registry with 2–3 repos; run `wsm create <ws> --repos ... --branch feature/x`.

- Execution:
  - Verify worktree directories exist.
  - Run `wsm delete <ws> --remove-files`.

- Assertions:
  - Worktrees present after create; absent after delete.
  - Use `WorktreeManager.List()` (via CLI verification output) to confirm.

- Backends: `hybrid`.

#### 15.6 Rebase (Happy Path)

- Files:
  - `test/integration/scenarios/rebase_happy_test.go`

- Setup:
  - Repo with upstream tracking; local commits need rebase.

- Execution:
  - `wsm sync pull --rebase`.

- Assertions:
  - Rebase completes; history linearized; ahead/behind zero when expected.

- Backends: `hybrid`.

#### 15.7 Rebase with Conflicts (Continue/Abort)

- Files:
  - `test/integration/scenarios/rebase_conflicts_test.go`

- Setup:
  - Create conflicting changes between local and upstream for the same files.

- Execution:
  - `wsm sync pull --rebase` → expect conflicts.
  - `wsm rebase status --repo <r>` → `stopped-conflicts`.
  - `wsm conflicts mark-resolved --repo <r> --all` and `wsm rebase continue` → expect completion.
  - Repeat on a fresh branch then `wsm rebase abort` → expect rollback.

- Assertions:
  - Status reflects in-progress and conflict states.
  - Continue leads to clean state; abort reverts.

- Backends: `hybrid`.

#### 15.8 Concurrency (`--jobs`)

- Files:
  - `test/integration/scenarios/jobs_test.go`

- Setup:
  - Workspace with 3–6 repos.

- Execution:
  - Run `status`, `diff`, `sync` with `--jobs=4`.

- Assertions:
  - No interleaving in final summaries (tables present and consistent).
  - Timings improved vs serial (coarse assertion or at least no failures).

- Backends: `hybrid`.

#### 15.9 Smoke Gate for PRs

- Files:
  - `test/integration/scenarios/smoke_test.go`

- Content:
  - Minimal test that builds a workspace with 1–2 repos, runs `status` and `diff`.

- Purpose:
  - Very fast gating in CI (run with `--smoke`).

---

### 16. How to Build and Run Tests (Step-by-Step)

1) Create helpers:
   - Add `test/integration/helpers/{sandbox.go,git.go,wsm.go}` with the APIs above.
   - Ensure `RunWSM` invokes the local built binary if present (`./.out/wsm`), else `go run ./cmd/wsm`.

2) Add first smoke test:
   - Create `test/integration/scenarios/status_diff_test.go` and implement the minimal scenario.
   - Locally run: `go test ./test/integration/... -v -count=1 -run 'Test(Smoke|Status|Diff)'`.

3) Run in Dagger:
   - `go run ./ci/dagger --backends=hybrid --smoke` (artifacts in `./.out`).

4) Expand scenarios incrementally:
   - Add commit+push, sync, worktree, rebase tests; validate locally first, then via Dagger.

5) Enable matrix in CI:
   - Add a GitHub Actions workflow invoking `go run ./ci/dagger` for smoke on PRs; nightly full matrix.

---
