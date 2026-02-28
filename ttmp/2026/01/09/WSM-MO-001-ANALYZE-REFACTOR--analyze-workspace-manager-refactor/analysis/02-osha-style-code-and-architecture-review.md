---
Title: OSHA-Style Code And Architecture Review (Exhaustive)
Ticket: WSM-MO-001-ANALYZE-REFACTOR
Status: active
Topics:
    - refactor
    - architecture
    - code-quality
    - correctness
    - feature-parity
    - workspace-manager
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: workspace-manager/cmd/wsm/root.go
      Note: Full command surface and root wiring
    - Path: workspace-manager/cmd/cmds/cmd_status.go
      Note: Workspace detection and shared helpers used across commands package
    - Path: workspace-manager/pkg/wsm/workspace.go
      Note: Largest core module; lifecycle orchestration and mixed abstraction usage
    - Path: workspace-manager/pkg/wsm/gitclient/hybrid_client.go
      Note: Hybrid fallback behavior; correctness-critical error propagation
    - Path: workspace-manager/pkg/wsm/gitclient/cli_client.go
      Note: Branch normalization behavior affecting remote branch detection
    - Path: workspace-manager/pkg/wsm/status.go
      Note: Status model population and currently disabled merged/rebase indicators
    - Path: workspace-manager/cmd/cmds/cmd_rebase.go
      Note: Command-level rebase implementation and ahead-count logic
    - Path: workspace-manager/cmd/cmds/cmd_pr.go
      Note: PR workflow assumptions on status data and direct GH CLI integration
    - Path: workspace-manager/cmd/cmds/cmd_push.go
      Note: Push candidate logic and remote-fallback behavior
    - Path: workspace-manager/test/integration/helpers/wsm.go
      Note: Test harness execution path and stale binary preference
    - Path: workspace-manager/test/integration/helpers/sandbox.go
      Note: Environment isolation gaps causing non-hermetic integration tests
    - Path: workspace-manager/README.md
      Note: User-facing command/build instructions drift
ExternalSources: []
Summary: Exhaustive quality and architecture inspection of current WSM implementation with severity-ranked findings, duplication/bloat analysis, and a concrete architecture expansion plan to preserve full current wsm feature parity during refactor.
LastUpdated: 2026-02-28T14:01:16-05:00
WhatFor: Provide a no-compromise quality baseline and migration plan for refactoring without losing current CLI capabilities.
WhenToUse: Use before major refactor phases, when triaging correctness risks, and when planning parity-safe architecture expansion.
---

# OSHA-Style Code And Architecture Review (Exhaustive)

## Executive Summary

This review inspected the `workspace-manager` codebase as if it were a safety-critical system under strict operational scrutiny: correctness first, then maintainability, then velocity. The key conclusion is that the project has strong feature breadth and useful abstractions started (`GitClient`, `WorktreeManager`, backend selection), but it is currently in a fragile mid-transition state where critical correctness behavior depends on implicit fallbacks and command-local shell orchestration.

The most severe blockers are not cosmetic:

1. `HybridClient` currently swallows real errors for several mutating operations, which can report success when operations failed.
2. Integration tests are not hermetic and may execute a stale prebuilt binary (`.out/wsm`) instead of current source, reducing trust in regressions and release safety.
3. Remote branch existence checks are semantically mismatched to backend behavior and can incorrectly report that remote branches do not exist.
4. Rebase metrics in command logic use reversed rev-list direction and can misreport “commits ahead”.

From an architecture perspective, the current refactor target needs to expand beyond Git backend abstraction to fully model the *actual* `wsm` feature set in the binary today (PR, push-to-fork workflows, merge orchestration, tmux/session automation, starship config generation). If these concerns are not formally represented in the target architecture, the refactor will either lose features or keep accumulating command-layer scripts and one-off behaviors.

## Scope And Inspection Standard

### Scope

- Reviewed all `cmd/wsm`, `cmd/cmds`, `pkg/wsm`, `pkg/wsm/gitclient`, `pkg/output` files.
- Reviewed integration test scenarios and harness in `test/integration`.
- Reviewed architecture docs (`README.md`, `IMPLEMENTATION.md`) for drift.
- Validated findings with line-level evidence and command-based inspection.

### Standard Applied

- Correctness over convenience.
- Architecture boundary integrity over tactical speed.
- No tolerance for hidden failure modes.
- Documentation must reflect executable reality.
- Test harnesses must be hermetic and trustworthy.

## Methodology

1. Surface inventory
- Enumerated command and package files; measured top file sizes.
- Enumerated command constructors and operational entry points.

2. Runtime and control-flow mapping
- Mapped CLI command paths into `pkg/wsm` services.
- Mapped backend calls to `BuildGitBackends` and direct shell invocations.

3. Hotspot scans
- Counted `exec.CommandContext` usage in command layer and core layer.
- Traced branch/remote detection semantics across backends.
- Traced rebase implementations across command and core.

4. Test validity checks
- Executed `go test ./...` and inspected failing scenarios.
- Traced test harness path/env behavior to determine whether tests validate current source.

5. Documentation and feature parity checks
- Compared documented build/install/usage patterns to current code and command registration.

## Current Feature Surface (As Shipped In `wsm`)

### Command surface from root wiring

Registered commands in [`cmd/wsm/root.go`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/root.go:58):

- `discover`, `list`, `create`, `fork`, `merge`, `add`, `remove`, `delete`, `info`, `status`, `pr`, `push`, `commit`, `sync`, `branch`, `rebase`, `diff`, `log`, `conflicts`, `tmux`, `starship`.

This is broader than the current “Git abstraction refactor” narrative. The architecture target must explicitly account for:

- SCM provider workflows (`gh`-based PR/push checks).
- Rebase/conflict lifecycle tools.
- Session automation (`tmux`).
- User environment mutation (`starship`).

### Size hotspots

Top file sizes from local scan:

- `pkg/wsm/workspace.go`: 1638 LOC
- `cmd/cmds/cmd_rebase.go`: 579 LOC
- `cmd/cmds/cmd_merge.go`: 469 LOC
- `pkg/wsm/discovery.go`: 406 LOC
- `cmd/cmds/cmd_status.go`: 393 LOC
- `pkg/wsm/sync_operations.go`: 380 LOC

The concentration in these files aligns with observed coupling and correctness drift.

## Severity-Ranked Findings

## Critical Finding 1: HybridClient can report success on failed operations

Problem:
`HybridClient` returns `nil` for mutating operations unless error equals `ErrNotImplemented`, which can suppress real failures from the primary backend.

Where to look:
- [`pkg/wsm/gitclient/hybrid_client.go:50`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/hybrid_client.go:50)
- [`pkg/wsm/gitclient/hybrid_client.go:54`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/hybrid_client.go:54)
- [`pkg/wsm/gitclient/hybrid_client.go:68`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/hybrid_client.go:68)
- [`pkg/wsm/gitclient/hybrid_client.go:72`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/hybrid_client.go:72)
- [`pkg/wsm/gitclient/hybrid_client.go:81`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/hybrid_client.go:81)
- [`pkg/wsm/gitclient/hybrid_client.go:85`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/hybrid_client.go:85)

Example snippet:

```go
func (h *HybridClient) Push(ctx context.Context, repo RepositoryHandle, remote string) error {
    if err := h.primary.Push(ctx, repo, remote); err == ErrNotImplemented { return h.fallback.Push(ctx, repo, remote) }
    return nil
}
```

Why it matters:
- False-success state is worse than explicit failure.
- Command flows (`commit --push`, `sync`, branch operations) can claim success while backend action failed.
- This undermines user trust and corrupts automation assumptions.

Cleanup sketch:

```go
func fallbackErr(err error) bool { return errors.Is(err, ErrNotImplemented) }

func (h *HybridClient) Push(...) error {
    err := h.primary.Push(...)
    if fallbackErr(err) {
        return h.fallback.Push(...)
    }
    return err
}
```

Do this consistently for `Add`, `Reset`, `Fetch`, `Push`, `CreateBranch`, `CheckoutBranch`.

## Critical Finding 2: Integration tests are non-hermetic and can run stale binaries

Problem:
Integration helper prefers module-root `.out/wsm` if present (currently present and old), and sandbox env preserves host `XDG_CONFIG_HOME`, defeating config isolation.

Where to look:
- Stale binary preference: [`test/integration/helpers/wsm.go:32`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/test/integration/helpers/wsm.go:32)
- Exec prebuilt binary: [`test/integration/helpers/wsm.go:37`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/test/integration/helpers/wsm.go:37)
- Env merge keeps host env: [`test/integration/helpers/sandbox.go:69`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/test/integration/helpers/sandbox.go:69)
- No `XDG_CONFIG_HOME` override: [`test/integration/helpers/sandbox.go:33`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/test/integration/helpers/sandbox.go:33)

Example snippet:

```go
bin := filepath.Join(moduleRoot, ".out", "wsm")
if _, err := os.Stat(bin); err == nil {
    cmd = exec.Command(bin, args...)
}
```

Observed execution evidence:
- `.out/wsm` exists in repo and is dated `Aug 24 2025`.
- `go test ./...` fails integration scenarios while `discover` reports implausible large repo counts (180+) in temp sandboxes, consistent with config leakage and stale executable behavior.

Why it matters:
- You can pass/fail tests against old code, not current source.
- Regression signals become noisy and can hide real breakage.
- This blocks safe refactoring.

Cleanup sketch:

```go
// sandbox default env
XDG_CONFIG_HOME = filepath.Join(home, ".config")
XDG_CACHE_HOME  = filepath.Join(home, ".cache")
XDG_DATA_HOME   = filepath.Join(home, ".local", "share")

// RunWSM
if forceGoRun || ciMode {
    cmd = exec.Command("go", "run", "./cmd/wsm", ...)
} else {
    cmd = exec.Command(filepath.Join(s.BaseDir, ".out", "wsm"), ...)
}
```

In CI, always build test-local binary from current source before scenarios.

## Critical Finding 3: Remote branch existence check is semantically broken

Problem:
`CheckRemoteBranchExists` expects `origin/<branch>` values, but CLI backend strips `remotes/origin/` and go-git returns local branches only.

Where to look:
- Consumer expectation: [`pkg/wsm/workspace.go:327`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workspace.go:327)
- Prefix expectation: [`pkg/wsm/workspace.go:329`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workspace.go:329)
- CLI normalization: [`pkg/wsm/gitclient/cli_client.go:57`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/cli_client.go:57)

Example snippet:

```go
if strings.HasPrefix(b, "origin/") && strings.TrimPrefix(b, "origin/") == branch {
    return true, nil
}
```

Why it matters:
- Worktree branch selection can incorrectly choose “new branch” paths.
- Fork/create/add flows may diverge from intended remote-tracking behavior.
- Behavior differs silently by backend mode.

Cleanup sketch:

```go
type GitClient interface {
    ListLocalBranches(...)
    ListRemoteBranches(...)
}

func CheckRemoteBranchExists(...) bool {
    remotes := gc.ListRemoteBranches(...)
    return contains(remotes, "origin/"+branch) || contains(remotes, branch)
}
```

## Critical Finding 4: Rebase “ahead” metric appears directionally reversed

Problem:
`cmd_rebase.go` computes commits with `HEAD..target`, which counts commits in target not in HEAD, yet the field is named `CommitsBefore/After` with “ahead” semantics.

Where to look:
- [`cmd/cmds/cmd_rebase.go:456`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_rebase.go:456)

Example snippet:

```go
cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("HEAD..%s", targetBranch))
```

Why it matters:
- User-facing reporting is misleading during rebase operations.
- Any decision logic built on this value risks inversion.

Cleanup sketch:

```go
// ahead of target
git rev-list --count <target>..HEAD
// behind target
git rev-list --count HEAD..<target>
```

Store both explicitly.

## High Finding 5: Rebase logic is duplicated at two abstraction levels

Problem:
There is a core rebase API (`pkg/wsm/rebase_operations.go`) and a separate command-level rebase executor in `cmd/cmds/cmd_rebase.go` with overlapping behavior.

Where to look:
- Core rebase API: [`pkg/wsm/rebase_operations.go:63`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/rebase_operations.go:63)
- Command-level rebase execution: [`cmd/cmds/cmd_rebase.go:439`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_rebase.go:439)

Why it matters:
- Behavior drift and bug-fix duplication.
- Conflicting conflict-detection semantics.
- Harder to test end-to-end consistency.

Cleanup sketch:
- Move `cmd_rebase` execution path to call core `wsm.Start/Status/Continue/Abort` only.
- Keep command layer as thin argument/parsing/formatting wrapper.

## High Finding 6: Status model fields exist but are effectively dead

Problem:
`IsMerged` and `NeedsRebase` exist in model and are consumed by commands, but are always set to `false` in status checker.

Where to look:
- Forced false assignment: [`pkg/wsm/status.go:143`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/status.go:143)
- PR command uses `IsMerged`: [`cmd/cmds/cmd_pr.go:242`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_pr.go:242)
- Utilities that could compute these are orphaned: [`pkg/wsm/git_utils.go:23`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/git_utils.go:23)

Why it matters:
- Misleading architecture contracts and potential future logic bugs.
- Command behavior depends on fields that do not reflect reality.

Cleanup sketch:
- Either compute these fields in status checker (using unified APIs), or remove them from public model and command logic.

## High Finding 7: Command layer still carries large operational shell logic

Problem:
Command files execute many git/gh/tmux shell commands directly, bypassing core services and making behavior hard to test and reason about.

Evidence:
- `cmd/cmds` has 25 direct `exec.CommandContext` calls.
- Highest counts: `cmd_rebase` (7), `cmd_push` (7), `cmd_pr` (7), `cmd_tmux` (3).

Where to look:
- Rebase shell execution: [`cmd/cmds/cmd_rebase.go:416`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_rebase.go:416)
- Push shell execution: [`cmd/cmds/cmd_push.go:298`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_push.go:298)
- PR shell execution: [`cmd/cmds/cmd_pr.go:286`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_pr.go:286)
- Tmux shell execution: [`cmd/cmds/cmd_tmux.go:199`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_tmux.go:199)

Why it matters:
- Violates layering objective.
- Inconsistent error handling and retry behavior.
- Hard to simulate in unit tests.

Cleanup sketch:
- Introduce service interfaces in `pkg/wsm`:
  - `SCMService` (GitHub/PR/push checks)
  - `RebaseService` (already partially present)
  - `SessionService` (`tmux` automation)
- Commands should call services, not shell commands directly.

## High Finding 8: Configuration architecture is split and partially inert

Problem:
Root initializes Viper (`clay.InitViper`) but core `loadConfig` uses hardcoded defaults and does not read Viper or documented env override for workspace dir.

Where to look:
- Viper init: [`cmd/wsm/root.go:51`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/wsm/root.go:51)
- Hardcoded workspace dir: [`pkg/wsm/workspace.go:432`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workspace.go:432)
- README env claim: [`README.md:284`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/README.md:284)

Why it matters:
- Operators cannot rely on documented configuration knobs.
- Test and runtime behavior differ unexpectedly.

Cleanup sketch:
- Centralize config in a `ConfigService` used by both command layer and core package.
- Resolve precedence explicitly: flags > env > config file > defaults.

## Medium Finding 9: Workspace detection logic is fragmented

Problem:
There are at least two workspace detection strategies: robust heuristic in `cmd_status.go` and simplified prefix match in `cmd_commit.go`.

Where to look:
- Heuristic resolver: [`cmd/cmds/cmd_status.go:88`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_status.go:88)
- Simplified resolver: [`cmd/cmds/cmd_commit.go:116`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_commit.go:116)

Why it matters:
- Commands can disagree on what “current workspace” means.
- User experience inconsistency across commands.

Cleanup sketch:
- Move detection into `pkg/wsm/workspace_context.go` with one canonical resolver.
- Reuse from all commands.

## Medium Finding 10: `workspace.go` has become a monolith

Problem:
`pkg/wsm/workspace.go` is 1638 LOC and mixes create/delete/orchestration, filesystem cleanup, metadata generation, setup-script discovery/execution, and branch/worktree decisions.

Where to look:
- File size and multi-domain responsibilities throughout [`pkg/wsm/workspace.go`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/workspace.go)

Why it matters:
- Harder to reason about correctness boundaries.
- Higher regression risk per edit.
- Slower onboarding and review throughput.

Cleanup sketch:

```text
pkg/wsm/
  workspace_service.go      # high-level orchestrator
  workspace_store.go        # save/load/list config
  worktree_lifecycle.go     # add/remove/rollback
  setup_scripts.go          # setup discovery/execution
  metadata.go               # .wsm/wsm.json handling
  workspace_context.go      # detect current workspace
```

## Medium Finding 11: Documentation drift is widespread and user-visible

Problem:
Docs still reference old binary path and old command naming.

Where to look:
- README build commands: [`README.md:45`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/README.md:45), [`README.md:501`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/README.md:501)
- `IMPLEMENTATION.md` stale layout and old paths: [`IMPLEMENTATION.md:39`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/IMPLEMENTATION.md:39)

Why it matters:
- Onboarding friction and broken copy/paste paths.
- Mismatch between architecture docs and executable code.

Cleanup sketch:
- Convert docs to a generated command reference section (from Cobra tree).
- Add CI doc drift checks for key command/build snippets.

## Medium Finding 12: Some operational rollback behavior is potentially destructive

Problem:
Merge rollback uses `git reset --hard origin/<base>` on repositories with prior successful merges.

Where to look:
- [`cmd/cmds/cmd_merge.go:439`](/home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/cmd/cmds/cmd_merge.go:439)

Why it matters:
- If assumptions are wrong, local changes can be destroyed.
- Rollback strategy should be explicit and auditable.

Cleanup sketch:
- Use temporary refs or merge snapshots before rollout.
- Prompt when hard reset is required unless explicit `--force-rollback-hard` flag.

## Duplication, Deprecation, And Bloat Assessment

### Duplication hotspots

1. Rebase operations duplicated in command and core.
2. Workspace detection duplicated with different semantics.
3. Branch/remote checks spread across command layer, core utils, and gitclient backends.
4. Build backend selection invoked repeatedly across hot paths instead of dependency-injected once.

### Deprecation and stale artifacts

- `README` and `IMPLEMENTATION` contain stale build paths and project layout references.
- Command examples still mention `workspace-manager` while binary is `wsm`.
- `pkg/output/Spinner` appears unused and implemented as busy loop without pacing.

### Bloat indicators

- Very broad command surface in one binary with mixed concerns (workspace orchestration, SCM PR automation, tmux automation, prompt config mutation).
- Large module dependency graph (`~490` dependencies reachable from `cmd/wsm` build), partly due monolithic module scope including CI tooling package.
- Multiple 100+ LOC functions carrying orchestration logic where service-level APIs should exist.

## Architecture Assessment: Current vs Needed Target

## What the current refactor target gets right

- Introduces `GitClient`/`WorktreeManager` seam.
- Supports backend modes (`gogit`, `cli`, `hybrid`) through `WSM_GIT_BACKEND`.
- Moves many status/sync/commit flows into core package abstractions.

## What it misses for full `wsm` parity

The current target architecture is Git-centric, but the shipped binary is product-centric and includes several non-Git domains:

- `PR` and push-to-fork orchestration (`gh` commands, remote checks, PR existence checks).
- Session automation (`tmux`) with profile-aware config discovery.
- Prompt configuration mutation (`starship`).
- Workspace context resolution shared by many commands.

If these domains stay command-local, the “single abstraction boundary” goal is incomplete.

## Expanded Target Architecture (Parity-Safe)

### Proposed service boundaries

```text
cmd/* (Cobra)
  -> App Services (pkg/wsm/services)
       - WorkspaceService
       - GitService
       - RebaseService
       - SCMService        # GitHub/PR/push workflows
       - SessionService    # tmux orchestration
       - PromptService     # starship config operations
       - WorkspaceContextResolver
       - ConfigService
  -> Adapters
       - git adapter (gogit/cli/hybrid)
       - gh adapter
       - tmux adapter
       - filesystem adapter
```

### Command-to-service mapping for feature parity

- `discover`, `list`, `info` -> `WorkspaceService` + `RepositoryCatalogService`
- `create`, `delete`, `add`, `remove`, `fork`, `merge` -> `WorkspaceService` + `WorktreeLifecycleService`
- `status`, `diff`, `commit`, `sync`, `branch` -> `GitService`
- `rebase`, `conflicts` -> `RebaseService`
- `push`, `pr` -> `SCMService`
- `tmux` -> `SessionService`
- `starship` -> `PromptService`

### Behavioral contracts that must be explicit

- Error propagation: no swallowed failures.
- Remote branch semantics: local vs remote APIs are separate and typed.
- Destructive operations: explicit policy and confirmation model.
- Workspace detection: one canonical resolver.
- Config precedence: deterministic and testable.

## Test And Verification Gaps

### Current gaps

- No unit tests in `pkg/wsm` and `pkg/wsm/gitclient` despite high-complexity orchestration.
- Integration scenarios do not cover many top-level commands (`fork`, `merge`, `push`, `pr`, `branch`, `tmux`, `starship`, `info`, `list`, `add`, `remove`).
- Integration harness currently risks validating stale binaries and host config state.

### Immediate test hardening

1. Fix harness hermeticity first (`XDG_*` and binary source).
2. Add backend contract tests for `gogit`, `cli`, and `hybrid` parity.
3. Add focused unit tests for:
- `HybridClient` error propagation behavior.
- remote branch detection semantics.
- workspace detection resolver.
- config precedence resolution.

## Recommended Cleanup And Migration Plan

## Phase 0 (Safety patch set, 1-2 days)

1. Fix `HybridClient` error propagation.
2. Fix integration harness hermeticity and stale-binary preference.
3. Patch remote branch detection contract/API.
4. Correct `rebase` ahead/behind metric direction.

Exit criteria:
- `go test ./...` stable on current source.
- No false-success path in hybrid mutators.

## Phase 1 (Boundary stabilization, 3-5 days)

1. Introduce `WorkspaceContextResolver` and replace command-local detection.
2. Introduce `ConfigService`; wire to root and core.
3. Move remaining `pkg/wsm` direct shell calls behind service adapters where practical.

Exit criteria:
- Command layer no longer contains workspace detection logic variants.
- Config behavior matches docs and is test-covered.

## Phase 2 (Parity architecture expansion, 1-2 weeks)

1. Add `SCMService` for `push`/`pr` workflows.
2. Add `SessionService` for `tmux` workflows.
3. Add `PromptService` for `starship` workflows (or explicitly mark as plugin/out-of-core).

Exit criteria:
- Command layer mostly delegates to service interfaces.
- PR/push behavior testable without direct command shelling in handlers.

## Phase 3 (Monolith split and maintainability, 1-2 weeks)

1. Decompose `workspace.go` into focused modules.
2. Remove orphan/dead fields or compute them properly (`IsMerged`, `NeedsRebase`).
3. Align docs with generated command/build references.

Exit criteria:
- Reduced file complexity and clearer ownership boundaries.
- Documentation and runtime are in sync.

## Open Questions For Architecture Decisions

1. Should `starship` remain in core binary or move to plugin/auxiliary tool?
2. Is `gh` mandatory for SCM flows, or should provider abstraction support non-GitHub remotes?
3. Should `merge` rollback remain hard-reset based, or switch to ref-based rollback snapshots?
4. Do we want strict “no direct exec in command layer” as an enforced lint rule after migration?

## Appendices

## Appendix A: Evidence summary (selected)

- Direct shell calls:
  - `cmd/cmds`: 25 occurrences of `exec.CommandContext`.
  - `pkg/wsm`: 27 occurrences of `exec.CommandContext`.
- Largest core file: `pkg/wsm/workspace.go` (1638 LOC).
- Largest command files: `cmd_rebase.go` (579), `cmd_merge.go` (469), `cmd_status.go` (393).
- Dependency breadth: `go list -deps ./cmd/wsm` resolves roughly 490 dependency nodes.

## Appendix B: Test run snapshot

Command executed:

```bash
go test ./...
```

Observed:
- Unit/package compilation succeeded for `cmd/*` and `pkg/*`.
- Integration scenario suite failed in multiple tests.
- Failure signatures consistent with non-hermetic test harness and stale binary preference:
  - workspace create failures with repo paths not existing in sandbox context.
  - anomalously large discovered repository counts in empty temp environments.

## Appendix C: Feature parity checklist for refactor completion

Required before declaring refactor “done” while preserving current `wsm` product surface:

- Workspace lifecycle parity (`create/fork/merge/add/remove/delete/info/list`).
- Git operation parity (`status/diff/commit/sync/branch/rebase/conflicts`).
- SCM workflow parity (`push/pr`).
- Session/tooling parity (`tmux/starship`) or explicit and documented scope reduction.
- Documentation parity (all examples execute as written with `wsm`).
- Hermetic integration and contract test coverage across backend modes.

---

This report is intentionally strict. The objective is not stylistic polish; it is operational safety, deterministic behavior, and a refactor path that preserves the full value users currently get from `wsm`.
