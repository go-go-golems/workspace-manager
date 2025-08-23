### 1. Purpose and Goals

This document specifies a comprehensive integration testing strategy for `workspace-manager` (WSM). It covers the scope of tests, scaffolding for deterministic environments, backend matrices, concurrency testing, rebase/conflict flows, and an optional Docker-based isolation to decouple tests from the developer machine.

---

### 2. Scope and Principles

- Exercise end-to-end flows through the public CLI (`wsm`) and (where helpful) targeted package-level helpers.
- Use hermetic, ephemeral sandboxes (temporary directories) with synthetic repositories and local bare remotes.
- Keep tests deterministic and parallel-safe; do not depend on external network services.
- Verify behavior across multiple WSM Git backends via `WSM_GIT_BACKEND` (`cli`, `gogit`, `hybrid`).
- Support concurrency testing with `--jobs` and ensure output and results are stable.

---

### 3. Environments and Matrices

- Backends: `cli`, `gogit`, `hybrid`.
- OS: primary Linux CI (Ubuntu), later macOS optional.
- Go versions: CI default toolchain (Go 1.24.x).
- Git versions: system-provided; tests must not rely on features beyond standard Git (>= 2.30 recommended).
- Matrices:
  - Minimal: run the full suite once for `hybrid`, and a reduced pass for `cli` and `gogit`.
  - Full (nightly): run the full suite for all three backends.

---

### 4. Test Data and Fixture Strategy

- Use helper builders to construct synthetic repos:
  - `initBareRepo(path)` → initialize a bare remote.
  - `initRepo(path, remoteURL)` → initialize a working repo with configured user/email and remote.
  - `commitFile(repoPath, filename, content, message)` → add and commit.
  - `createBranch(repoPath, name, base)` and `checkout(repoPath, name)`.
  - `introduceConflict(repoPathA, repoPathB, filename)` → create divergent changes.
- Workspaces:
  - For `worktree` flows, create a “registry” that points to several repos; `wsm create` to materialize worktrees under a temp workspace root.
- Output capture:
  - Capture CLI stdout/stderr; parse only essentials to avoid brittle assertions.
- Timeouts:
  - Use per-test context with a reasonable timeout (e.g., 30–90s), fail on timeout.

---

### 5. Test Scenarios (Core)

- Status:
  - Clean workspace, modified files, staged files, untracked files.
  - Ahead/behind against local bare remote (simulate fetch/push).
  - Backend parity (sanity) for `cli` vs `gogit` vs `hybrid`.
- Diff:
  - Unified diff across multiple repos; `--staged` and full diff; repo filter.
  - Concurrency via `--jobs` to ensure ordering remains readable.
- Commit:
  - `commit --add-all` across multiple repos; message templating; `--push` to local bare remotes.
  - Verify commits landed and refs moved as expected.
- Sync:
  - `sync pull` (fetch-only fast path) and `sync push`; ahead/behind before/after checks.
  - `sync all --dry-run` prints a stable summary; with/without changes.
  - `--jobs` parallelism does not interleave final tables; counts are consistent.
- Worktree:
  - `create` workspace with multiple repos; verify worktrees exist and metadata is written.
  - `delete` with proper worktree removal; use `WorktreeManager.List()` verification.

---

### 6. Rebase and Conflict Scenarios

- Rebase happy path:
  - Create repo with local commits; set upstream (bare remote); `sync pull --rebase` performs rebase (no conflicts).
  - Validate history shape post-rebase and ahead/behind equals zero when expected.
- Rebase with conflicts:
  - Diverge local and upstream on the same files; `sync pull --rebase` stops with conflicts.
  - `wsm rebase status` shows `stopped-conflicts` and lists conflicted files.
  - Use `wsm conflicts open --repo <r>` (non-blocking in CI; can validate exit code) and `wsm conflicts mark-resolved --repo <r> --all`, then `wsm rebase continue` completes.
  - `wsm rebase abort` rolls back to pre-rebase state.
- Rebase manual mode:
  - `wsm rebase --manual` prints suggested commands and does not modify repos.

---

### 7. Concurrency Tests

- `status`, `diff`, `sync` with `--jobs > 1` across 2–6 repos.
- Verify no data races with `-race` in CI (unit-level where possible; integration smoke for high-level flows).
- Ensure final summaries (tables) remain stable and readable; results correspond to real repo states.

---

### 8. CLI vs API Coverage

- Primary: drive through `wsm` CLI for end-to-end behavior.
- Secondary: where stability demands, use exported helpers (e.g., fixture builders, light-weight client calls) to assert on internal states without duplicating logic.

---

### 9. Scaffolding in Repository

- Directory layout (proposal):
  - `test/integration/`
    - `helpers/` → repo/workspace builders, git utilities, tempdir mgmt, env setup (HOME, GIT config).
    - `scenarios/` → test files grouped by feature (status_diff_sync_test.go, commit_test.go, worktree_test.go, rebase_test.go).
    - `testdata/` → templates for files to commit (optional; may generate on the fly).
  - `Makefile` targets:
    - `make test-integration` → `go test ./test/integration/... -v -count=1`.
    - `make test-integration-race` → add `-race`.
    - `make test-integration-backends` → run with matrix `WSM_GIT_BACKEND`.

- Test harness conventions:
  - Set `HOME` to a temp dir and initialize `~/.gitconfig` with test user/email.
  - Set `WSM_GIT_BACKEND` per test or subtest.
  - Build `wsm` binary once per package (or call `go run ./cmd/wsm`) and invoke via `exec.CommandContext`.
  - Ensure all temp dirs are cleaned at the end (defer cleanup; print on failure for debugging).

---

### 10. Example Test Skeletons

- Go test (status + diff):
```go
func TestStatusAndDiff(t *testing.T) {
    t.Parallel()
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    env := NewSandbox(t) // sets HOME, PATH, temp dirs
    defer env.Cleanup()

    remote := env.InitBareRepo("remote.git")
    repo1 := env.InitRepo("repo1", remote)
    repo2 := env.InitRepo("repo2", remote)

    env.CommitFile(repo1, "a.txt", "hello", "add a")
    env.CommitFile(repo2, "b.txt", "world", "add b")

    ws := env.CreateWorkspace("ws1", []RepoSpec{{Name: "repo1", Path: repo1}, {Name: "repo2", Path: repo2}})

    out := env.RunWSM(ctx, "status", "--workspace", ws.Name)
    require.Contains(t, out.Stdout, "Workspace:")

    diff := env.RunWSM(ctx, "diff", "--workspace", ws.Name)
    require.Contains(t, diff.Stdout, "=== Repository: repo1 ===")
}
```

- Go test (rebase with conflicts):
```go
func TestRebaseWithConflicts(t *testing.T) {
    t.Parallel()
    ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
    defer cancel()

    env := NewSandbox(t)
    defer env.Cleanup()

    // Setup remote and two clones to create divergence
    remote := env.InitBareRepo("remote.git")
    repo := env.InitRepo("repo", remote)

    // upstream change
    env.CommitFileToRemote(remote, "f.txt", "upstream", "upstream commit")
    env.RunGit(ctx, repo, "fetch", "origin")

    // local conflicting change
    env.CommitFile(repo, "f.txt", "local", "local conflicting commit")

    // Create workspace containing the repo as a worktree
    ws := env.CreateWorkspace("ws", []RepoSpec{{Name: "repo", Path: repo}})

    // This should attempt fetch+rebase and stop on conflict
    run := env.RunWSM(ctx, "sync", "pull", "--workspace", ws.Name, "--rebase")
    require.NotZero(t, run.ExitCode)

    st := env.RunWSM(ctx, "rebase", "status", "--repo", "repo")
    require.Contains(t, st.Stdout, "stopped-conflicts")

    // Mark resolved and continue (simplify by choosing --all)
    env.RunWSM(ctx, "conflicts", "mark-resolved", "--repo", "repo", "--all")
    env.RunWSM(ctx, "rebase", "continue")
}
```

---

### 11. Docker-Based Isolation (Optional)

Goal: run integration tests inside an isolated container, removing dependencies on the host’s Git configuration and filesystem.

- Image contents:
  - Base: `golang:1.24-bullseye` (or similar).
  - Packages: `git`, common mergetools (optional), `bash`, `diffutils`.
  - Working dir: `/workspace` where the repo is mounted.

- Dockerfile (example):
```dockerfile
FROM golang:1.24-bullseye
RUN apt-get update && apt-get install -y --no-install-recommends git bash ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
ENV CGO_ENABLED=0
# cache go mod
COPY go.mod go.sum ./
RUN go mod download
# copy rest and build tests/binary in container
COPY . .
RUN go build -o /usr/local/bin/wsm ./cmd/wsm
CMD ["bash"]
```

- Running tests in Docker:
  - Bind mount the repo and run `go test`:
    - `docker build -t wsm-test .`
    - `docker run --rm -v "$PWD":/workspace -w /workspace -e WSM_GIT_BACKEND=hybrid wsm-test bash -lc "go test ./test/integration/... -v -count=1"`
  - For matrices: vary `WSM_GIT_BACKEND` in separate runs.

- Isolation details:
  - Set `HOME=/tmp/wsm-home` inside container; create `.gitconfig` with test user/email.
  - GitHub CLI or external services are not required; avoid network calls.

- Makefile targets:
```make
.PHONY: test-integration-docker
test-integration-docker:
	docker build -t wsm-test .
	docker run --rm -v "$(PWD)":/workspace -w /workspace \
	  -e WSM_GIT_BACKEND=hybrid wsm-test bash -lc 'go test ./test/integration/... -v -count=1'
```

---

### 12. Logging, Debuggability, and Artifacts

- On failure, print sandbox paths and retain temp directories (disable cleanup) to allow post-mortem.
- Write captured command outputs and logs to files under the sandbox (e.g., `/tmp/wsm-it/<test-name>/cmd.log`).
- Gate verbose logging via env var (e.g., `WSM_TEST_VERBOSE=1`).

---

### 13. Risks and Mitigations

- Flaky timing (e.g., concurrent output ordering):
  - Assert summaries, not live stream interleaving; collect per-repo results before printing.
- Git version differences:
  - Avoid relying on presentation formats that changed across versions; focus on semantic results (exit code, ref states).
- Leftover files:
  - Always clean sandbox unless test failed and retention is enabled.

---

### 14. Actionable TODOs

- [ ] Add `test/integration/helpers` with sandbox and git utility functions.
- [ ] Add initial scenarios for status/diff/sync/commit/worktree.
- [ ] Add rebase/conflict scenarios (happy path + conflicts + abort/continue).
- [ ] Introduce backend matrix runs in CI for `cli`, `gogit`, and `hybrid`.
- [ ] Optional: Dockerfile + `make test-integration-docker` target.
- [ ] Add retention-on-failure and logging utilities.
