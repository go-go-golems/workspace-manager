---
Title: Investigation Diary
Ticket: WSM-MO-013-FORK-REBASE-STATUS
Status: active
Topics:
    - workspace-manager
    - git
    - rebase
    - fork
    - bugfix
    - status
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/wsm/branch/status_resolve.go
      Note: ResolveBaseRef resolver + BaseRefResolution (Step 4, commit a58504b)
    - Path: repo://pkg/wsm/branch/status_resolve_test.go
      Note: ResolveBaseRef unit tests (Step 4, commit a58504b)
    - Path: repo://pkg/wsm/git_utils.go
      Note: honest CheckBranchMerged/CheckBranchNeedsRebase returning BaseComparison (Step 5, commit ba6b6f7)
    - Path: repo://pkg/wsm/git_utils_test.go
      Note: forked-workspace regression tests (Step 5, commit ba6b6f7)
    - Path: repo://pkg/wsm/status.go
      Note: getRepositoryStatusWithClient wires status.Base + bool mirrors (Step 5, commit ba6b6f7)
    - Path: repo://pkg/wsm/types.go
      Note: BaseComparison model + type aliases to branch (Step 4, commit a58504b)
ExternalSources: []
Summary: Chronological investigation of the forked-workspace rebase/merge status bug, from log triage to root cause and proposed fix.
LastUpdated: 2026-08-19T10:32:00-04:00
WhatFor: Resuming or reviewing the investigation of WSM-MO-013.
WhenToUse: Read before continuing work on the forked-workspace status bug.
---







# Investigation Diary

## Goal

Record how the `wsm status` forked-workspace bug was triaged, root-caused, and
scoped into a design + implementation plan in ticket WSM-MO-013-FORK-REBASE-STATUS.

## Step 1: Triage the user-reported log and locate the failing code

The user ran `wsm status` in the forked workspace
`/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql` and pasted debug logs
showing, per sub-repo, `merged=false` followed by
`Failed to check for commits ahead on configured remote base error="exit status 128"`
with `upstream=origin/task/deploy-dev-indexer`.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket for handling the fact that wsm cannot apparently figure out the rebase merge status when a wsm project has been forked, for exampke in /home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql" (plus the debug log block and "when I do wsm status").

**Assistant interpretation:** Create a docmgr ticket investigating why WSM cannot determine rebase/merge status for forked workspaces; reproduce and root-cause it.

**Inferred user intent:** Stop `wsm status` from silently misreporting `IsMerged`/`NeedsRebase` for forked workspaces, and document the fix for a new engineer.

### What I did
- Searched the codebase for the log strings: `rg -n "Failed to check for commits ahead" pkg` → `pkg/wsm/git_utils.go:91`.
- Confirmed the two check functions live in `pkg/wsm/git_utils.go` (`CheckBranchMerged`, `CheckBranchNeedsRebase`).
- Located the call site: `pkg/wsm/status.go:151-157` (`getRepositoryStatusWithClient`).

### Why
The log strings are the fastest path from symptom to source. The `exit status 128`
plus `upstream=origin/<base>` immediately suggested a missing git ref.

### What worked
- Direct string search pinpointed the exact file and lines in one query.
- Reading `git_utils.go` showed both checks hardcode `origin/<base>` via
  `branchsvc.RemoteTrackingRef(branchsvc.DefaultRemoteName, base)`.

### What didn't work
- Nothing failed; the investigation was straightforward.

### What I learned
- The merge check (`git_utils.go:42-52`) discards the command error and returns
  `(false, nil)`, so a missing ref looks like "not merged".
- The rebase check (`git_utils.go:85-92`) returns the error, which the caller
  swallows via an `err == nil` guard, leaving `NeedsRebase` at default `false`.

### What was tricky to build
N/A (investigation only; no code changed).

### What warrants a second pair of eyes
- Confirm that the two checks really diverge in error handling (merge swallows,
  rebase returns) — this asymmetry is the crux of why the symptom is two-faced.

### What should be done in the future
- Unify the two checks' error contract in the fix (Step 3 design).

### Code review instructions
- Start at `pkg/wsm/git_utils.go:24` (`CheckBranchMerged`) and `:56`
  (`CheckBranchNeedsRebase`).
- Validate by grepping the log strings against the source.

### Technical details
- `exit status 128` = git "unknown revision / not a valid object name".

## Step 2: Reproduce against the real repo and confirm root cause

### What I did
- Inspected `/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql/.wsm/wsm.json`:
  `baseBranch: task/deploy-dev-indexer`, `branch: task/ragkit-coinvault-mysql`.
- Ran the exact failing commands in the worktree `.../geppetto`:
  - `git rev-list --count HEAD..origin/task/deploy-dev-indexer` → `fatal: ... unknown revision`, exit 128.
  - `git merge-base --is-ancestor HEAD origin/task/deploy-dev-indexer` → `fatal: Not a valid object name`, exit 128.
- Checked ref existence:
  - `git for-each-ref refs/remotes/origin/task/deploy-dev-indexer` → **empty** (remote-tracking ref does NOT exist).
  - `git for-each-ref refs/heads/task/deploy-dev-indexer` → **exists locally** (`335a807a...`).
- Checked remotes: `origin` = `go-go-golems/geppetto`, `wesen` = `wesen/geppetto`. The base branch was never pushed to `origin`.

### Why
Prove the assumption `origin/<base>` exists is false for this workspace, and that
the local base branch *does* exist (so a local fallback is viable).

### What worked
- The ref-existence checks are decisive and reproducible: remote-tracking ref
  absent, local branch present.

### What didn't work
- One bash command broke on the `%(refname)` format string with unquoted parens;
  re-ran without the format string.

### What I learned
- The base branch `task/deploy-dev-indexer` is a **local-only task branch**.
  Forking from a local-only base is the trigger condition.
- `ForkWorkflow.Plan` (`pkg/wsm/workflows/fork_workflow.go:71-88`) detects the
  base branch as the source workspace's *current branch*, which is frequently a
  local task branch — so forks of in-flight work inherently lack `origin/<base>`.

### What was tricky to build
N/A.

### What warrants a second pair of eyes
- Verify the fork-detects-base-from-current-branch claim holds for other forked
  workspaces, not just this one.

### What should be done in the future
- Consider a `wsm fork` warning when the detected base has no remote-tracking ref.

### Code review instructions
- Reproduce with the commands above in `.../ragkit-coinvault-mysql/geppetto`.

### Technical details
- `git for-each-ref <refspec>` prints nothing when the ref is absent; non-zero
  output means it exists.

## Step 3: Map the surrounding system and design the fix

### What I did
- Read the four-layer architecture doc (`pkg/docs/04-architecture-overview.md`).
- Read the status call chain: `workflows/status_workflow.go` → `status.go`
  (`StatusChecker`, `getRepositoryStatusWithClient`) → `git_utils.go`.
- Read the branch-resolution subsystem: `pkg/wsm/branch/{types,resolver,service,service_impl}.go`
  and the `gitclient.GitClient` interface + CLI impls of
  `RemoteTrackingBranchExists`/`LocalBranchExists`/`AheadBehind`.
- Confirmed the `branch` package already has a `BranchResolutionPlan` with
  strategies (`UseLocal`, `TrackRemote`, …) and existence-based resolution, but
  the two status checks bypass it.
- Wrote the design doc (`design-doc/01-…`) with executive summary, evidence,
  decision records, pseudocode, phased plan, test strategy, and file references.

### Why
The fix should reuse the existing resolution abstraction rather than add ad-hoc
git heuristics in the workflow layer. Documenting the whole pipeline makes the
ticket intern-friendly as requested.

### What worked
- The `branch` subsystem cleanly supports a `ResolveBaseRef` helper: prefer
  `origin/<base>` (remote-tracking), fall back to local `<base>`, else "unknown".

### What didn't work
- Nothing.

### What I learned
- The asymmetry between the two checks (swallow vs return) is what produces the
  two-faced symptom (`merged=false` confidently + `exit status 128` as an error).
- The caller's `err == nil` guard is what turns "unknown" into a confident
  `false` for `NeedsRebase`.

### What was tricky to build
- Designing the "unknown" state without breaking the existing JSON `bool`
  fields (D2): keep `is_merged`/`needs_rebase` as `bool` defaulting `false`, add
  an internal tri-state + an optional `base_ref`/`base_ref_status` column.

### What warrants a second pair of eyes
- The signature change `(value, resolved bool, err)` propagating through the
  call site; ensure concurrency in `GetWorkspaceStatusWithOptions` stays safe
  (it does — resolution is read-only).

### What should be done in the future
- Implement Phases 1–6 of the design doc; add the regression test
  `TestStatus_ForkedWorkspace_LocalOnlyBase`.

### Code review instructions
- Read `design-doc/01-forked-workspace-rebase-merge-status-bug-analysis-and-implementation-guide.md` §5–§8.
- Validate proposed `ResolveBaseRef` against `pkg/wsm/branch/resolver.go`.

### Technical details
- Reuse `gitclient.RemoteTrackingBranchExists` (`cli_client.go:107`) and
  `LocalBranchExists` (`cli_client.go:53`), both `for-each-ref` based.

## Step 4: Phase E1 part 1 — BaseComparison model + ResolveBaseRef resolver

This step landed the foundation for honest merge/rebase status: a typed
`BaseComparison` outcome on `RepositoryStatus`, and a `ResolveBaseRef` helper
in the `branch` package that picks a concrete comparison ref by preferring the
remote-tracking branch and falling back to the local branch. Nothing is wired
into the live status path yet, so there is no behavior change — but the types
and the resolver are in place and unit-tested.

**Commit (code):** a58504b — ":sparkles: Add BaseComparison model and ResolveBaseRef resolver"

### Prompt Context

**User prompt (verbatim):** (see Step 1) — the user later asked to implement the first set of features (E1-E6) before the fork one, commit at appropriate intervals, and keep a detailed diary.

**Assistant interpretation:** Build the base-resolution foundation (Phase E1) first so later phases (default detection, per-repo override, set-base, status table) and the fork-divergence work can lean on it. Commit in focused steps with a diary.

**Inferred user intent:** A correct, honest, well-tested core that the rest of the ticket builds on; nothing merged until it compiles and tests pass.

### What I did
- `pkg/wsm/types.go`: added `BaseComparison` struct (ConfiguredBase, Remote, ResolvedRef, RefSource, Status, Reason, IsMerged, NeedsRebase) and made `BaseComparisonStatus`/`RefSource` **type aliases** of `branch.BaseResolutionStatus`/`branch.RefSource` so the resolution layer stays the single source of truth. `RepositoryStatus` gains `Base BaseComparison`; `IsMerged`/`NeedsRebase` kept as bool mirrors for JSON compat.
- `pkg/wsm/branch/status_resolve.go`: new `ResolveBaseRef(ctx, gc, repoPath, base, remote) (BaseRefResolution, error)` — prefers `RemoteTrackingBranchExists` (`<remote>/<base>`), falls back to `LocalBranchExists` (`<base>`), else `BaseUnknown` with a precise reason; `BaseError` + reason on git failure. Also added `DistinctBranches` + `MostFrequentBranch` helpers (for the later fork-divergence prompt).
- `pkg/wsm/branch/status_resolve_test.go`: tests for remote-tracking-preferred, local fallback, unknown-when-absent, empty base, default-remote normalization, and the helpers.

### Why
The status checks today conflate "git errored" with "the answer is false" and hardcode `origin/<base>`. The fix needs (a) a provenance-bearing outcome and (b) a reusable resolver that already exists conceptually in `branch` but wasn't used. Putting the enums in `branch` (not `wsm`) avoids an import cycle and keeps policy in the branch layer.

### What worked
- Type aliases (`type BaseComparisonStatus = branch.BaseResolutionStatus`) gave one source of truth without breaking the JSON contract or forcing `wsm` to redeclare constants.
- Reusing the `createRemoteBranchFixture` shape from `remote_branch_exists_test.go` made the resolver tests fast and self-contained.

### What didn't work
- First test run failed: the fixture tried to create the local-only base branch from the seed's `task/deploy-dev-indexer` commit, but the client (cloned with `--branch main`) didn't have that object → `fatal: not a valid branch point`. Fixed by branching from the client's own `main` tip instead (`git branch task/deploy-dev-indexer main`), which is the real-world shape anyway (the base exists locally, just not remotely).
- Parent `go.work` said `go 1.25` while the module requires `go 1.26.4`; `go build` refused. Fixed with `go work edit -go 1.26.4` + `GOTOOLCHAIN=auto` (downloads 1.26.4 on demand). `GOWORK=off` also works for the Makefile's lint target.

### What I learned
- A `git clone --branch main` does NOT fetch other branches' objects, so you cannot point a local branch at a commit that only existed on an un-pushed branch in the seed. Branch from something the client already has.
- Named string types (`type RemoteName string`) convert cleanly with `string(r)` but have no inherent `String()` method; use explicit conversion in calls.

### What was tricky to build
- Keeping one source of truth for the status enums across two packages without an import cycle: `branch` may not import `wsm`. Solved with type aliases in `wsm` pointing at `branch`. A naive duplicate-declaration would compile but drift.
- The empty-base case: `ResolveBaseRef("")` returns `BaseUnknown` ("base branch is empty"). That is correct for the resolver, but the *checks* must first run `branch.ResolveBaseBranch("")` (→ env → "main") so an empty workspace `BaseBranch` still compares against `main`. (Bitten by this in Step 5.)

### What warrants a second pair of eyes
- The type-alias decision: confirm `wsm.BaseResolved == branch.BaseResolved` is the same value (it is, by alias) and that JSON marshaling of `BaseComparisonStatus` still emits "resolved"/"unknown"/"error" (it does — the underlying type is `string`).

### What should be done in the future
- Once E5 (status table) lands, the `BASE` column should read `Base.ResolvedRef` + `Base.RefSource`; ensure the alias values render as the documented strings.
- Consider moving `DistinctBranches`/`MostFrequentBranch` to a shared spot if fork-divergence (F1) needs them from the `workflows` package (they're in `branch`, importable).

### Code review instructions
- Start at `pkg/wsm/branch/status_resolve.go` (`ResolveBaseRef`), then `pkg/wsm/types.go` (`BaseComparison`, the alias block).
- Validate: `GOTOOLCHAIN=auto go test ./pkg/wsm/branch/ ./pkg/wsm/ -count=1` → ok.

### Technical details
- Preference order in `ResolveBaseRef`: `RemoteTrackingBranchExists` → `LocalBranchExists` → `BaseUnknown`. Mirrors `branch.resolveFromState` but inverted (remote first for status, since it reflects upstream truth).
- Fixture: bare `origin.git` + seed on `main` (pushed) + `task/deploy-dev-indexer` (NOT pushed) + client cloned `--branch main` + local `task/deploy-dev-indexer` branched from `main`.

## Step 5: Phase E1 part 2 — honest checks + status wiring + regression tests

This step rewrote `CheckBranchMerged`/`CheckBranchNeedsRebase` to use
`ResolveBaseRef` and return `BaseComparison`, wired them into
`getRepositoryStatusWithClient`, and added regression tests proving the
forked-workspace case now resolves via the local fallback instead of swallowing
an `exit 128`. This is the step that actually fixes the user-reported symptom.

**Commit (code):** ba6b6f7 — ":sparkles: Wire honest merge/rebase checks into status (Phase E1 part 2)"

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Finish E1: make the checks honest (resolved/unknown/error instead of a swallowed false), wire the call site, and prove it with tests.

**Inferred user intent:** The actual bug fix — `wsm status` on a forked workspace must stop reporting a confident false and stop logging `exit status 128`.

### What I did
- `pkg/wsm/git_utils.go`: both checks now take `(ctx, gc gitclient.GitClient, path, baseBranch, remote)` and return `(BaseComparison, error)`. They run `branch.ResolveBaseBranch(baseBranch)` first (so empty → main), then `branch.ResolveBaseRef`. `merge-base --is-ancestor` exit 1 ("not an ancestor") is distinguished from a real failure via `errors.As(*exec.ExitError)`; stderr is captured into `Reason` via new `runGitCapture`/`runGitCaptureNoOut` helpers (no bare exit codes). `pkg/errors` aliased `pkgerrors` to free the `errors` name for stdlib `errors.As`.
- `pkg/wsm/status.go`: `getRepositoryStatusWithClient` calls the new checks with the `gc` it already holds, sets `status.Base`, and keeps `IsMerged`/`NeedsRebase` as bool mirrors.
- `pkg/wsm/git_utils_test.go`: regression tests — forked-workspace local fallback (`BaseResolved` via local), unknown-when-absent (`BaseUnknown` + reason), remote-tracking-resolved (common case), skip-when-on-base.

### Why
The old `merged := err == nil` turned "ref doesn't exist" into `false`; the old rebase check returned the error which the caller's `if err == nil` swallowed into `false`. Both lied. The new contract: `Status` tells the truth; `IsMerged`/`NeedsRebase` are only meaningful when `Status == BaseResolved`.

### What worked
- `errors.As(*exec.ExitError)` cleanly separated merge-base's "not an ancestor" (exit 1) from a genuine failure (exit 128), which the old `err == nil` could not.
- Reusing the `gc` already held by `getRepositoryStatusWithClient` avoided adding a new dependency and kept the existence checks on the same backend as the rest of status.

### What didn't work
- First attempt at `isNotAncestorExit` used `github.com/pkg/errors.Unwrap` + an interface assertion; `pkg/errors.Wrapf` returns a wrapper whose Unwrap chain didn't match my ad-hoc interface, so exit 1 propagated as an error and tests failed. Fixed by aliasing `pkg/errors` as `pkgerrors` and using stdlib `errors.As(*exec.ExitError)`.
- `TestStatusSemanticMergedAndNeedsRebase` (integration) went red: the workspace had no `--base-branch`, so `BaseBranch == ""`, and my new checks passed `""` straight to `ResolveBaseRef` → `BaseUnknown` → `needs_rebase=false` where the test expected `true` after `--fetch`. Root cause: I dropped the `branch.ResolveBaseBranch("")` → `main` fallback that the old code had. Fixed by calling `branch.ResolveBaseBranch(baseBranch)` in both checks before `ResolveBaseRef`.

### What I learned
- "Empty base" and "base doesn't exist" are different: empty means "use the default (main)"; missing means "unknown". The resolver handles missing; the *checks* must normalize empty → default first. This is exactly the kind of precedence bug the design doc's §5.6 warns about, and it's a hint that E3's `ResolveBaseBranchForRepo` should centralize ALL precedence (including the empty→main fallback) so callers can't forget it.
- `merge-base --is-ancestor` exits 0 (ancestor/merged) or 1 (not ancestor); any other code is a real error. Treating "non-zero" as "not merged" silently corrupts status on a broken repo.

### What was tricky to build
- The `pkg/errors` vs stdlib `errors` naming collision: the file historically imported `github.com/pkg/errors` as `errors`. Adding stdlib `errors.As` required aliasing `pkg/errors` to `pkgerrors` and updating every `errors.Wrapf` call in the file — easy to miss one and get a build break.
- Preserving the "skip rebase when on the base branch" optimization through the signature change: the skip compares `currentBranch` against the *resolved* base (`string(base)` after `ResolveBaseBranch`), not the raw `baseBranch`, so `main`/`master` equivalence still works.

### What warrants a second pair of eyes
- The `status.go` merge of the two `BaseComparison`s: it sets `status.Base` from the merged check, then overlays `NeedsRebase` from the rebase check, and promotes a rebase `BaseError` over a merged `BaseResolved`. Confirm this precedence (error beats resolved) is the desired semantics — an alternative is to keep two separate fields. Right now `Base` carries one `Status` for both checks, which is correct because both compare against the same resolved ref.
- The `errors.As` unwrap path: confirm it holds when `runGitCaptureNoOut` wraps with `pkgerrors.Wrapf` (it does — `errors.As` walks the unwrap chain regardless of which errors package wrapped it).

### What should be done in the future
- E3 should introduce `ResolveBaseBranchForRepo` and have BOTH checks call it, removing the per-check `ResolveBaseBranch` call and the risk of one forgetting it. The empty→main fallback must live there.
- E5 should render `Base.Status` in the table (`✓`/`-`/`?`/`!`) so a human sees `unknown` rather than a misleading `false`.
- Add a test for the `BaseError` path (e.g. a corrupted/non-git repo) once E5 surfaces it; currently only `BaseUnknown` and `BaseResolved` are covered.

### Code review instructions
- Start at `pkg/wsm/git_utils.go` (`CheckBranchMerged`, `CheckBranchNeedsRebase`, `isNotAncestorExit`), then `pkg/wsm/status.go` (`getRepositoryStatusWithClient` end).
- Validate: `GOTOOLCHAIN=auto go test ./... -count=1` → all ok (unit + `test/integration/scenarios`).

### Technical details
- `isNotAncestorExit`: `var ee *exec.ExitError; errors.As(err, &ee); ee.ExitCode() == 1`.
- `runGitCapture` uses `cmd.CombinedOutput()` so stderr lands in the wrapped error message (the reason `BaseError.Reason` is human-readable).
- The integration test `TestStatusSemanticMergedAndNeedsRebase` is the canary for the empty-base→main fallback; if it breaks again, the checks are passing `""` to `ResolveBaseRef` instead of `ResolveBaseBranch("")` first.
