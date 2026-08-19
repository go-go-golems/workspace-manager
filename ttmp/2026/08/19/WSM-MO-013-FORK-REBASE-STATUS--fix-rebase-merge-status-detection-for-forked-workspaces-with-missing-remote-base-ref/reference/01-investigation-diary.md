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
    - Path: repo://cmd/wsm/cmds/workspace/set_base.go
      Note: wsm set-base command default/--global (E4, commit 925c327)
    - Path: repo://pkg/wsm/branch/status_resolve.go
      Note: |-
        ResolveBaseRef resolver + BaseRefResolution (Step 4, commit a58504b)
        DefaultBaseBranchForRepo + ResolveBaseBranchForRepo precedence (E2/E3, commits aae5359/f842186)
    - Path: repo://pkg/wsm/branch/status_resolve_test.go
      Note: ResolveBaseRef unit tests (Step 4, commit a58504b)
    - Path: repo://pkg/wsm/git_utils.go
      Note: honest CheckBranchMerged/CheckBranchNeedsRebase returning BaseComparison (Step 5, commit ba6b6f7)
    - Path: repo://pkg/wsm/git_utils_test.go
      Note: forked-workspace regression tests (Step 5, commit ba6b6f7)
    - Path: repo://pkg/wsm/gitclient/cli_client.go
      Note: DefaultBranch via symbolic-ref (E2, commit aae5359)
    - Path: repo://pkg/wsm/status.go
      Note: |-
        getRepositoryStatusWithClient wires status.Base + bool mirrors (Step 5, commit ba6b6f7)
        getRepositoryStatusWithClient uses ResolveBaseBranchForRepo (E3, commit f842186)
    - Path: repo://pkg/wsm/types.go
      Note: BaseComparison model + type aliases to branch (Step 4, commit a58504b)
    - Path: repo://pkg/wsm/workspace.go
      Note: RepositoryMetadata overrides + overlayWorkspaceBaseOverrides + SetRepoBase (E3/E4, commits f842186/925c327)
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

## Step 6: Phase E2 — discover per-repo default base branch via symbolic-ref

This step stopped the hardcoded "main" assumption. Each repo's remote-advertised
default branch is now discovered (`git symbolic-ref refs/remotes/origin/HEAD`),
persisted on the repo and in `.wsm/wsm.json`, and a shared helper combines
symbolic-ref with a main/master/develop probe fallback. Status doesn't consume
it yet (E3 wires precedence); this step just makes the data available.

**Commit (code):** aae5359 — ":sparkles: E2: discover per-repo default base branch via symbolic-ref (Q3)"

### Prompt Context

**User prompt (verbatim):** (see Step 4) — then "do E2 - E4, commit at appropriate intervals."

**Assistant interpretation:** Implement default-branch detection (Q3), per-repo override + precedence (Q4 core), and the set-base command (Q4 surface), committing each phase separately.

**Inferred user intent:** A repo like goldeneaglecoin.com (origin/HEAD -> develop) must compare against develop, not main; the foundation for "manage different branches" per the user's earlier note.

### What I did
- `pkg/wsm/gitclient/client.go`: added `DefaultBranch(ctx, repo, remote)` to the `GitClient` interface.
- `pkg/wsm/gitclient/cli_client.go`: implemented via `git symbolic-ref refs/remotes/<remote>/HEAD`, stripped to the branch name; unset HEAD returns `("", nil)` (not an error) so callers probe candidates.
- `pkg/wsm/gitclient/default_branch_test.go`: advertised (develop), unset-returns-empty (deletes `origin/HEAD` to simulate a truly unset remote — `git clone --branch main` *synthesizes* `origin/HEAD`, so the delete is necessary), default-remote normalization.
- `pkg/wsm/branch/status_resolve.go`: `DefaultBaseBranchForRepo` combines symbolic-ref with a `main`/`master`/`develop` probe via `RemoteTrackingBranchExists`.
- `pkg/wsm/types.go`: `Repository.DefaultBaseBranch` (json `default_base_branch`).
- `pkg/wsm/discovery.go`: `analyzeRepository` populates it.
- `pkg/wsm/workspace.go`: `RepositoryMetadata.DefaultBaseBranch` (json `defaultBaseBranch`); emitted in `createWorkspaceMetadata`.

### Why
The remote's own advertised default is the most correct base; probing covers the rare unset-HEAD case. Persisting it on the repo + in `.wsm/wsm.json` means status can use it without re-running git on every status call.

### What worked
- `symbolic-ref` on the bare remote *before* push (`git --git-dir remoteA.git symbolic-ref HEAD refs/heads/develop`) reliably advertises `develop` as the default, so a normal `git clone` sets `origin/HEAD -> origin/develop`. Verified the test returns `"develop"`.

### What didn't work
- First `TestCliDefaultBranch_UnsetReturnsEmpty` failed: `git clone --branch main remoteB clientB` set `origin/HEAD -> origin/main` on the client even though the bare remote's HEAD was never set (clone synthesizes it from the cloned branch). Fixed by deleting the ref on the client: `git symbolic-ref -d refs/remotes/origin/HEAD` — that's the only way to simulate a genuinely unset default.

### What I learned
- `git clone --branch X` always sets `origin/HEAD -> origin/X` on the client, regardless of the remote's own HEAD. So "unset default" is hard to construct from a clone alone; you must delete the synthesized ref.
- A repo can legitimately default to `master` or `develop` (goldeneaglecoin.com confirmed `origin/HEAD -> origin/develop` during the design phase), so the probe order `main, master, develop` is a documented heuristic, not "main always".

### What was tricky to build
- Keeping the "unset HEAD is not an error" contract (`("", nil)`) distinct from a real git failure, so the probe fallback runs only when symbolic-ref genuinely has nothing. `runGit` returns an error on non-zero exit, so the impl swallows that specific error and returns empty.

### What warrants a second pair of eyes
- The probe order (`main, master, develop`) is arbitrary-but-documented; confirm no repo in the fleet defaults to something else (e.g. `trunk`, `production`). If so, extend the list or prefer the symbolic-ref result strictly.

### What should be done in the future
- E3's `ResolveBaseBranchForRepo` consumes `DefaultBaseBranch` (done in Step 7); a follow-up could re-discover it on `wsm discover` only (cheap to store), not on every status — currently it's persisted, so status doesn't re-run git for it.

### Code review instructions
- Start at `pkg/wsm/gitclient/cli_client.go` (`DefaultBranch`), then `pkg/wsm/branch/status_resolve.go` (`DefaultBaseBranchForRepo`), then `pkg/wsm/discovery.go` (`analyzeRepository`).
- Validate: `GOTOOLCHAIN=auto go test ./pkg/wsm/gitclient/ -run DefaultBranch -v`.

### Technical details
- `git symbolic-ref refs/remotes/origin/HEAD` -> `refs/remotes/origin/develop` -> trim prefix -> `"develop"`.

## Step 7: Phase E3 — per-repo override + precedence + LoadWorkspace overlay

This step wired the discovered default into a single precedence resolver and
added per-repo overrides with local-beats-global semantics. All base resolution
now flows through `ResolveBaseBranchForRepo`, which structurally prevents the
empty→main bug from Step 5 recurring (the checks can no longer forget a layer).

**Commit (code):** f842186 — ":sparkles: E3: per-repo base override + precedence + LoadWorkspace overlay (Q4)"

### What I did
- `pkg/wsm/branch/status_resolve.go`: `RepoBaseInput` + `ResolveBaseBranchForRepo` implementing 6-layer precedence (in-workspace per-repo > config-dir per-repo > workspace base > discovered default > env > main). Returns `(branch, remote)`; override `BaseRemote` wins, defaulting to origin.
- `pkg/wsm/types.go`: `Repository` gains `BaseBranch`/`BaseRemote` (config-dir, `--global`; json `base_branch`/`base_remote`) and `BaseBranchWorkspace`/`BaseRemoteWorkspace` (in-workspace overlay; `json:"-"` since they come from `.wsm/wsm.json` at load).
- `pkg/wsm/workspace.go`: `RepositoryMetadata` gains `BaseBranch`/`BaseRemote` (in-workspace `.wsm/wsm.json` fields). New `overlayWorkspaceBaseOverrides` reads `<workspace>/.wsm/wsm.json` and overlays per-repo overrides onto the loaded `Repository` as the `*Workspace` fields (local beats global). Called from both `LoadWorkspace` and `LoadWorkspaces` (the latter is what `wsm status` uses). Missing/unreadable `wsm.json` is non-fatal.
- `pkg/wsm/status.go`: `getRepositoryStatusWithClient` builds a `RepoBaseInput` from the repo (with overlaid fields) + workspace base, calls `ResolveBaseBranchForRepo`, passes `(base, remote)` to the checks. Replaces the per-check `ResolveBaseBranch` call.
- Tests: `precedence_test.go` (all 6 layers + override-remote + env-vs-discovered + a `TestMain` that unsets `WSM_BASE_BRANCH` so `MainFallback` isn't flaky), `workspace_overlay_test.go` (overlay merge, missing-file safety, nil safety).

### Why
Centralizing precedence in one function means the empty→main fallback (which bit Step 5) lives in exactly one place. Two stores with flag selection (not mirroring) keeps each write to one file and makes precedence auditable.

### What worked
- Type aliases + a `RepoBaseInput` struct in `branch` (not `wsm`) avoided the import cycle while letting `wsm` build the input from loaded types.
- The overlay reads the in-workspace file the user actually sees, so editing `.wsm/wsm.json` by hand works without a re-save.

### What didn't work
- First overlay test asserted `BaseRemoteWorkspace` empty for `goldeneaglecoin.com`, but the fixture set `BaseRemote: "origin"`, so it was correctly `"origin"`. Fixed the expectation (the overlay faithfully copies whatever's in the file, including explicit "origin").

### What I learned
- `json:"-"` on `BaseBranchWorkspace`/`BaseRemoteWorkspace` keeps them out of the config-dir JSON (they're load-time overlays, not persisted there), while still letting `LoadWorkspace` set them on the in-memory struct.
- `TestMain` in a package test file is the clean way to neutralize env vars (`WSM_BASE_BRANCH`) that would make a "falls back to main" test flaky if inherited from the environment.

### What was tricky to build
- The overlay must run in BOTH `LoadWorkspace` and `LoadWorkspaces`, because `wsm status` goes through `WorkspaceContextService.LoadWorkspace` → `LoadWorkspaces`. Forgetting one makes the override invisible to status. Caught by the integration suite still passing (it uses the default path), but the overlay test asserts both.
- Distinguishing "no override file" (non-fatal, inherit config-dir) from "corrupt file" (log a debug warning, inherit) — both fall through rather than failing status.

### What warrants a second pair of eyes
- The `status.go` merge of two `BaseComparison`s (merged sets `Base`, rebase overlays `NeedsRebase`, rebase `BaseError` promotes over merged `BaseResolved`). Confirm error-beats-resolved is desired; both checks compare the same ref so one `Status` is correct, but if they ever diverge (e.g. ref deleted between the two calls) the promotion is the safer choice.
- `BaseBranchWorkspace`/`BaseRemoteWorkspace` use `json:"-"` so they're NEVER persisted to config-dir — confirm no code path serializes a `Repository` expecting them to round-trip (they're rebuild-only at load).

### What should be done in the future
- E5 should render `Base.Status` + `Base.ResolvedRef`/`RefSource` in the table so the precedence winner is visible (e.g. "develop (default)" vs "task/x (workspace)").
- Consider a `wsm show-base` / `wsm status --verbose` that prints the resolved precedence layer per repo, to make the 6-layer order auditable at runtime.

### Code review instructions
- Start at `pkg/wsm/branch/status_resolve.go` (`ResolveBaseBranchForRepo`), then `pkg/wsm/workspace.go` (`overlayWorkspaceBaseOverrides`), then `pkg/wsm/status.go` (`getRepositoryStatusWithClient`).
- Validate: `GOTOOLCHAIN=auto go test ./pkg/wsm/branch/ ./pkg/wsm/ -count=1`.

### Technical details
- Precedence is a `switch` (first non-empty wins); `normalizeRemote` (already in resolver.go) handles empty→origin for the override remotes.

## Step 8: Phase E4 — wsm set-base command (default in-workspace, --global config-dir)

This step added the user-facing command to set a per-repo base override,
writing to one of two stores (never both), with optional `--fetch` to
materialize the remote-tracking ref.

**Commit (code):** 925c327 — ":sparkles: E4: wsm set-base command (default in-workspace, --global config-dir)"

### What I did
- `pkg/wsm/workspace.go`: `SetRepoBase(ctx, workspaceName, SetRepoBaseOptions)` + `SetRepoBaseOptions{RepoName, Branch, Remote, Global, Fetch}`. Validates the repo exists; `Global=true` writes `Repository.BaseBranch`/`BaseRemote` via `SaveWorkspace`; default writes `RepositoryMetadata.BaseBranch`/`BaseRemote` to `.wsm/wsm.json` via new `setRepoBaseInWorkspace` (preserves all other metadata). Optional `--fetch` runs `git fetch <remote> <branch>` (`fetchBaseRef`); fetch failure is non-fatal (warns, still records the override).
- `cmd/wsm/cmds/workspace/set_base.go`: `SetBaseCommand` (Bare + Glaze dual mode) with flags `--branch` (required), `--remote`, `--global`, `--fetch`, plus `--workspace` and positional `<repo-name>`. Detects workspace from cwd. Prints `[stored: workspace|global]`.
- `cmd/wsm/cmds/workspace/root.go`: register `set-base`.
- Tests: `set_base_test.go` — default-writes-in-workspace-only (config-dir untouched), `--global`-writes-config-dir-only (`.wsm/wsm.json` untouched), local-beats-global-after-load (overlay precedence), required-branch, unknown-repo.

### Why
Mirroring to both stores creates two sources of truth that drift; flag-selection keeps each write to one file. Defaulting to the in-workspace store (the most local, highest-precedence) means the common "set a base for this worktree" needs no flag. `--fetch` makes the override immediately usable (the remote-tracking ref exists) without a separate step.

### What worked
- Reusing `WorkspaceManager.LoadWorkspace`/`SaveWorkspace` for the `--global` path kept the config-dir write consistent with how workspaces are normally persisted.
- The "fetch failure is non-fatal" choice lets a user set a base for a not-yet-pushed branch and still record the override; status resolves once the ref exists.

### What didn't work
- Several `declared and not used` build failures from multi-return captures (`wm, wsPath, configPath := ...`) where a test only used some returns. Go is strict; fixed by discarding with `_` at the call site and recomputing the in-workspace path from the loaded `Workspace.Path` where needed.

### What I learned
- For test helpers returning `(wm, wsPath, configPath)`, only capture what each test uses; recomputing from a loaded object (e.g. `loaded.Path`) is cleaner than carrying an unused variable.
- A command that writes two different stores based on a flag should validate the target store's repo entry exists in BOTH the workspace repos and (for the default path) the in-workspace metadata — `setRepoBaseInWorkspace` errors if the repo isn't in `.wsm/wsm.json`, which is the right guard (the file is the source of truth for that store).

### What was tricky to build
- The default path writes `.wsm/wsm.json` while the `--global` path writes config-dir JSON; the command must not accidentally write both. Enforced by the `if opts.Global { ... return }` early return so the in-workspace write only runs in the default branch.
- `--fetch` runs `git fetch` in the worktree path, which must exist; `fetchBaseRef` stats it first and returns a clear error if the worktree is missing (e.g. repo not added to the workspace).

### What warrants a second pair of eyes
- `SetRepoBase` loads the workspace via `wm.LoadWorkspace` (which overlays in-workspace overrides), but for `--global` it then mutates `Repository.BaseBranch` and saves — confirm the overlay fields (`BaseBranchWorkspace`, `json:"-"`) are NOT serialized into the config-dir JSON by `SaveWorkspace` (they're `json:"-"`, so they aren't). If they were, a re-save would leak in-workspace values into the config-dir store.
- The `--fetch` warning on failure: confirm it doesn't mask a genuinely broken remote (e.g. wrong remote name). The override is still recorded, so the user can fix the remote and re-run status; acceptable.

### What should be done in the future
- A `--clear` flag to remove an override (revert to inherited). Trivial extension: set the field to `""` and save.
- E5 should surface the resolved base + store in `wsm status` so a user can see which store a given repo's base came from.

### Code review instructions
- Start at `pkg/wsm/workspace.go` (`SetRepoBase`, `setRepoBaseInWorkspace`, `fetchBaseRef`), then `cmd/wsm/cmds/workspace/set_base.go` (`execute`).
- Validate: `GOTOOLCHAIN=auto go test ./pkg/wsm/ -run SetRepoBase -v` and `go run ./cmd/wsm set-base --help`.

### Technical details
- `wsm set-base <repo> --branch develop --fetch` → writes `.wsm/wsm.json` RepositoryMetadata.BaseBranch="develop", then `git fetch origin develop` in `<ws>/<repo>`.
- `wsm set-base <repo> --branch develop --global` → writes config-dir `Repository.BaseBranch="develop"` via `SaveWorkspace`; `.wsm/wsm.json` untouched.
