---
Title: Status Reporting Enhancements — Per-Repo Base, Default Detection, and Honest Comparison Results
Ticket: WSM-MO-013-FORK-REBASE-STATUS
Status: active
Topics:
    - workspace-manager
    - git
    - rebase
    - fork
    - bugfix
    - status
    - api-design
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql/goldeneaglecoin.com
      Note: repo with origin/HEAD -> develop (Q3 evidence)
    - Path: repo://cmd/wsm/cmds/workspace/status.go
      Note: table gains BASE column and honest MERGED/REBASE (Q1/Q2)
    - Path: repo://pkg/wsm/branch/types.go
      Note: ResolveBaseBranch extends to per-repo precedence (Q3/Q4)
    - Path: repo://pkg/wsm/discovery.go
      Note: analyzeRepository populates DefaultBaseBranch (Q3)
    - Path: repo://pkg/wsm/git_utils.go
      Note: CheckBranchMerged/NeedsRebase rewrite to return BaseComparison (Q1/Q2)
    - Path: repo://pkg/wsm/gitclient/cli_client.go
      Note: runGit captures stderr (Q1); add DefaultBranch via symbolic-ref (Q3)
    - Path: repo://pkg/wsm/types.go
      Note: RepositoryStatus to carry BaseComparison; Repository to carry DefaultBaseBranch
    - Path: repo://pkg/wsm/workspace.go
      Note: RepositoryMetadata gains BaseBranch/BaseRemote; LoadWorkspace merges .wsm/wsm.json overrides (E3)
ExternalSources: []
Summary: 'Intern-oriented analysis, design, and implementation guide for four status-reporting enhancements: (1) show why a merge/rebase comparison failed, (2) show which branch the status was computed against, (3) discover the default base branch per repo instead of assuming main, and (4) allow a per-worktree override of the rebase comparison branch.'
LastUpdated: 2026-08-19T11:00:00-04:00
WhatFor: Onboarding an engineer to the four status-reporting enhancements that build on the WSM-MO-013 base-resolution fix.
WhenToUse: Read this before extending pkg/wsm/status.go, pkg/wsm/discovery.go, pkg/wsm/types.go, or the status table formatter.
---









# Status Reporting Enhancements — Per-Repo Base, Default Detection, and Honest Comparison Results

> Audience: a new engineer (intern-level). This document assumes you have read
> the companion doc `01-forked-workspace-rebase-merge-status-bug-analysis-and-implementation-guide.md`
> (the base bug analysis). If not, read at least its §3 (architecture) and §4
> (gap analysis) first; this guide builds directly on the `ResolveBaseRef`
> concept introduced there.
>
> This guide explains the four enhancements requested after the base bug was
> scoped. They are tightly related — all four share one data model — so we
> design them together.

## 1. Executive Summary

The base fix in `01-...` makes `wsm status` *correct* (compare against a ref
that exists) and *honest* (say "unknown" when no ref exists). This document
extends that work along four axes the user asked for:

1. **Q1 — Why did the comparison fail?** Surface a human-readable reason in the
   status report instead of a silent `false`.
2. **Q2 — Which branch was status computed against?** Show the concrete ref used
   (e.g. `origin/develop` vs `task/deploy-dev-indexer (local)`) in the report.
3. **Q3 — Per-repo default base branch.** Stop assuming `main`. Some repos use
   `master`, some use `develop` (verified: `goldeneaglecoin.com` defaults to
   `origin/develop`). Discover the remote's advertised default via
   `git symbolic-ref refs/remotes/origin/HEAD`.
4. **Q4 — Per-worktree override.** Let a user set, for a specific worktree in a
   workspace, which branch the rebase/merge check should compare against — even
   when it differs from the workspace-level base.

All four are unified by a single new struct, `BaseComparison`, attached to each
`RepositoryStatus`. The struct records *what* was compared, *how* it was
resolved, and *why* if it could not be. The status table gains a `BASE` column
and stops printing confident booleans when the truth is "unknown".

## 2. Problem Statement and Scope

### 2.1 What the base fix leaves on the table

After `01-...`, `wsm status` would resolve a base ref (prefer remote-tracking,
fall back to local, else "unknown") and would not swallow git errors. But four
things remain unsatisfactory:

1. When the comparison *fails* (no ref, or git errors), the user sees `false`
   with no explanation. They cannot tell "not merged" from "couldn't tell".
2. The user cannot see *which* ref the merge/rebase answer was computed against,
   so they cannot sanity-check it (e.g. "am I really comparing against
   `develop`, not `main`?").
3. The default base branch is a hardcoded `"main"` (`pkg/wsm/branch/types.go:20`
   `DefaultBaseBranch`). Repositories whose default is `master` or `develop`
   are silently compared against the wrong branch.
4. There is one workspace-level `BaseBranch` and no way to override it per
   worktree. A mixed workspace (some repos on `main`, one on `develop`) cannot
   express that.

### 2.2 In scope

- The `BaseComparison` data model and its rendering in the status table.
- `DefaultBranch` discovery + persistence in the registry and `wsm.json`.
- A per-repo `BaseBranch`/`BaseRemote` override stored in `wsm.json`.
- A new `wsm set-base` command to set the override.
- The resolution precedence that ties Q3 + Q4 together.

### 2.3 Out of scope

- The base bug itself (covered by `01-...`).
- Changing the public JSON shape of existing `is_merged`/`needs_rebase` fields
  (they stay `bool` for compatibility; new fields are additive).
- The rebase-in-progress detection in worktrees (separate ticket
  WSM-MO-012).

## 3. Current-State Architecture (the parts you need)

This section focuses only on the pieces the four enhancements touch. Read the
companion doc for the full status pipeline.

### 3.1 The two places a workspace's data lives (critical subtlety)

A workspace is stored in **two** JSON files, and they are not automatically kept
in sync. This is the most important fact for Q4.

1. **Config-dir JSON** — `~/.config/workspace-manager/workspaces/<name>.json`
   (`pkg/wsm/workspace.go:449-475`, `LoadWorkspaces`; `:516-528`,
   `LoadWorkspace`). This is the **canonical** store that `wsm status` actually
   loads via `WorkspaceContextService.LoadWorkspace`
   (`pkg/wsm/workspace_context.go:21-35`), which calls `LoadWorkspaces`.

2. **In-workspace JSON** — `<workspace>/.wsm/wsm.json`
   (`pkg/wsm/workspace.go:1509-1528`, `WorkspaceMetadata`/`RepositoryMetadata`).
   This is written by `createWorkspaceMetadata` (`:1530-1547`) and is the file a
   user *sees* when they `ls` their workspace, and the one they would naturally
   edit.

```go
// pkg/wsm/workspace.go:1522-1528  (in-workspace per-repo metadata — today)
type RepositoryMetadata struct {
    Name         string   `json:"name"`
    Path         string   `json:"path"`
    Categories   []string `json:"categories"`
    WorktreePath string   `json:"worktreePath"`
}
```

> **Consequence for Q4:** A per-repo override that only lives in `.wsm/wsm.json`
> would be **invisible** to `wsm status`, because status loads from the
> config-dir JSON. We resolve this with **two distinct override stores** and a
> precedence rule: `wsm set-base` (default) writes only the in-workspace
> `.wsm/wsm.json`; `wsm set-base --global` writes only the config-dir workspace
> JSON. At load time the in-workspace override wins over the config-dir
> override (local beats global). See Decision E3.

### 3.2 The status record and its bool fields

```go
// pkg/wsm/types.go:46-58
type RepositoryStatus struct {
    ...
    IsMerged       bool       `json:"is_merged"`    // True if branch is merged to the default remote main branch
    NeedsRebase    bool       `json:"needs_rebase"` // True if branch needs to be rebased on the default remote main branch
}
```

Note the comments literally say "default remote **main** branch" — this encodes
both the Q3 bug (assumes main) and the Q1/Q2 gap (no provenance). These fields
have no information about *which* ref they were computed against or whether the
computation even succeeded.

### 3.3 How the comparison result reaches the UI

```go
// pkg/wsm/status.go:151-157  (the call site — base fix changes these guards)
if isMerged, err := CheckBranchMerged(ctx, repoPath, baseBranch); err == nil {
    status.IsMerged = isMerged
}
if needsRebase, err := CheckBranchNeedsRebase(ctx, repoPath, baseBranch); err == nil {
    status.NeedsRebase = needsRebase
}
```

```go
// cmd/wsm/cmds/workspace/status.go  (the table)
// statusToRows emits is_merged / needs_rebase columns (line ~196)
// printStatusDetailed prints MERGED / REBASE columns (line ~250) via
// getMergedString / getRebaseString, which return "✓"/"-"/"⚠️".
func getMergedString(status wsm.RepositoryStatus) string {
    if status.IsMerged { return "✓" }
    return "-"
}
```

So the data path is: `CheckBranchMerged` → `RepositoryStatus.IsMerged` (bool) →
`statusToRows` (JSON) / `getMergedString` (table). To carry Q1/Q2 information,
every hop needs the richer struct.

### 3.4 Base branch resolution today (the Q3 root cause)

```go
// pkg/wsm/branch/types.go:19-36
const DefaultBaseBranch BranchName = "main"

func ResolveBaseBranch(explicit string) BranchName {
    if strings.TrimSpace(explicit) != "" { return BranchName(...) }
    if v := os.Getenv("WSM_BASE_BRANCH"); v != "" { return BranchName(v) }
    return DefaultBaseBranch   // ← "main", always
}
```

`ResolveBaseBranch` is called in `pkg/wsm/git_utils.go:24` and `:57`. It takes
*one* explicit string (the workspace `BaseBranch`) and falls back to `"main"`.
There is no per-repo input and no concept of "the remote's default branch".

### 3.5 Discovery does not record a default branch

```go
// pkg/wsm/discovery.go:182-201  (analyzeRepository)
if handle, err := gc.Open(ctx, path); err == nil {
    if remoteURL, err := gc.RemoteURL(ctx, handle, "origin"); err == nil {
        repo.RemoteURL = remoteURL
    }
    if branch, err := gc.CurrentBranch(ctx, handle); err == nil { ... }
    ...
}
```

Discovery grabs `RemoteURL`, `CurrentBranch`, `Branches`, `Tags`, `LastCommit`
— but **no default branch**. The `Repository` struct
(`pkg/wsm/types.go:7-17`) has no field for it. So even if we detected it, there
is nowhere to put it.

### 3.6 The git primitives we will reuse / add

Existing (`pkg/wsm/gitclient/client.go` interface + `cli_client.go` impls):

```go
RemoteTrackingBranchExists(ctx, repo, remote, branch) (bool, error)  // for-each-ref refs/remotes/<r>/<b>
LocalBranchExists(ctx, repo, branch) (bool, error)                  // for-each-ref refs/heads/<b>
RemoteURL(ctx, repo, remote) (string, error)
```

New (Q3): `DefaultBranch` — detect via `git symbolic-ref refs/remotes/<remote>/HEAD`.

The shared shell-out helper that all CLI methods use (`pkg/wsm/gitclient/cli_client.go:30`):

```go
func runGit(ctx context.Context, dir string, args ...string) ([]byte, error) {
    cmd := exec.CommandContext(ctx, "git", args...)
    cmd.Dir = dir
    out, err := cmd.CombinedOutput()   // ← note: captures stderr, unlike git_utils.go
    ...
}
```

`runGit` already returns stderr in its error — which is exactly what Q1 needs to
surface a real reason. The legacy `git_utils.go` uses `exec.CommandContext`
directly and discards stderr; the fix moves comparisons onto `runGit`-style
capturing.

## 4. Gap Analysis (evidence-backed)

### 4.1 Q3 evidence — default branch is not always `main`

Verified on the actual repos in the failing workspace:

```
$ cd .../goldeneaglecoin.com
$ git symbolic-ref refs/remotes/origin/HEAD
refs/remotes/origin/develop          ← default is "develop", not "main"
$ git branch -r | grep -E 'origin/(main|master|develop)$'
  origin/HEAD -> origin/develop
  origin/develop
  origin/master                       ← also has master

$ cd .../geppetto
$ git symbolic-ref refs/remotes/origin/HEAD
refs/remotes/origin/main             ← geppetto defaults to "main"
```

So assuming `main` is correct for `geppetto` but **wrong** for
`goldeneaglecoin.com` (would compare against a non-default `main`/`master`
instead of `develop`). `git symbolic-ref refs/remotes/origin/HEAD` is the
reliable, remote-advertised answer; it is unset only when the remote never
published a HEAD (rare; then we probe candidates).

### 4.2 Q1 evidence — reasons are discarded today

- `pkg/wsm/git_utils.go:42-44`: `merged := err == nil` — exit 128 ("not a valid
  object name") becomes `merged=false` with no reason.
- `pkg/wsm/git_utils.go:85`: `cmd.Output()` discards stderr; only `err` survives,
  and the caller (`status.go:155`) swallows it.

So for the failing workspace the user sees `merged=false` and an `exit status
128` debug log, with no indication that the *cause* is "the configured base ref
does not exist remotely or locally".

### 4.3 Q2 evidence — no provenance in the record

`RepositoryStatus` carries `IsMerged bool` and `NeedsRebase bool` and nothing
else. There is no field naming the ref used. The table columns `MERGED`/`REBASE`
(`status.go:250`) therefore cannot show what they were computed against.

### 4.4 Q4 evidence — single workspace-level base, no override

`Workspace.BaseBranch` (`pkg/wsm/types.go:31`, json `base_branch`) is the only
base. It flows into every repo uniformly (`status.go:152` passes the same
`baseBranch` to all repos). `RepositoryMetadata` (`workspace.go:1523-1528`) has
no per-repo base field, and there is no command to set one.

## 5. Proposed Architecture and APIs

### 5.1 The unifying data model (Q1 + Q2)

A single struct captures the entire base-comparison outcome and rides on
`RepositoryStatus`. Existing `IsMerged`/`NeedsRebase` bools are kept for JSON
compatibility and are simply mirrors of the struct's values.

```go
// pkg/wsm/types.go (new)

// BaseComparisonStatus classifies the outcome of resolving the comparison ref.
type BaseComparisonStatus string

const (
    BaseResolved BaseComparisonStatus = "resolved" // a real ref was compared
    BaseUnknown  BaseComparisonStatus = "unknown"  // no usable ref exists
    BaseError    BaseComparisonStatus = "error"    // git itself failed
)

// BaseComparison is the provenance-bearing result of a merge/rebase check.
type BaseComparison struct {
    // ConfiguredBase is what the workspace/config said to compare against
    // (after precedence resolution: per-repo override > workspace > discovered
    // default > env > "main"). E.g. "task/deploy-dev-indexer" or "develop".
    ConfiguredBase string `json:"configured_base"`

    // Remote is the remote used (default "origin", overridable per repo).
    Remote string `json:"remote"`

    // ResolvedRef is the concrete git ref compared against: "origin/main",
    // "origin/develop", or "task/deploy-dev-indexer" (local fallback).
    // Empty when Status != resolved.
    ResolvedRef string `json:"resolved_ref"`

    // RefSource is how ResolvedRef was found: "remote-tracking" | "local" | "".
    RefSource string `json:"ref_source"`

    // Status classifies the outcome (resolved/unknown/error).
    Status BaseComparisonStatus `json:"base_status"`

    // Reason is a human-readable explanation when Status != resolved.
    Reason string `json:"reason,omitempty"`

    // IsMerged / NeedsRebase are meaningful only when Status == resolved.
    IsMerged    bool `json:"is_merged"`
    NeedsRebase bool `json:"needs_rebase"`
}
```

`RepositoryStatus` gains:

```go
type RepositoryStatus struct {
    ...
    IsMerged    bool            `json:"is_merged"`     // mirror, kept for compat
    NeedsRebase bool            `json:"needs_rebase"`  // mirror, kept for compat
    Base        BaseComparison  `json:"base"`          // NEW: provenance + values
}
```

### 5.2 ResolveBaseRef returns a structured outcome (Q1)

The companion doc's `ResolveBaseRef(ref, found, err)` becomes richer, carrying
the reason and source. This is the single place that classifies outcomes.

```go
// pkg/wsm/branch (new file, e.g. status_resolve.go)

type BaseRefResolution struct {
    Ref    string                 // concrete ref, "" if not resolved
    Source string                 // "remote-tracking" | "local" | ""
    Status BaseComparisonStatus   // resolved | unknown | error
    Reason string                 // human explanation when !resolved
}

// ResolveBaseRef picks the concrete ref to compare HEAD against.
// Preference: <remote>/<base> (remote-tracking) → <base> (local) → unknown.
// On a genuine git failure (e.g. corrupt repo), returns Status=error with the
// captured stderr in Reason.
func ResolveBaseRef(
    ctx context.Context,
    gc gitclient.GitClient,
    repoPath string,
    base branch.BranchName,
    remote branch.RemoteName,
) (BaseRefResolution, error)
```

Pseudocode:

```text
func ResolveBaseRef(gc, repoPath, base, remote):
    if remote == "": remote = "origin"
    h = gc.Open(repoPath)
    if err: return {Status: error, Reason: "open: " + err}

    # 1) prefer remote-tracking ref
    ok, err = gc.RemoteTrackingBranchExists(h, remote, base)
    if err:  return {Status: error, Reason: stderr(err)}   # capture stderr
    if ok:   return {Ref: remote+"/"+base, Source: "remote-tracking", Status: resolved}

    # 2) fall back to local base branch
    ok, err = gc.LocalBranchExists(h, base)
    if err:  return {Status: error, Reason: stderr(err)}
    if ok:   return {Ref: base, Source: "local", Status: resolved}

    # 3) nothing to compare against — give a precise reason
    return {Status: unknown,
            Reason: base+" is not a remote-tracking ref on "+remote+
                    " and is not a local branch"}
```

Concrete reasons this produces (Q1):

- *Forked workspace, local fallback works:* `Ref="task/deploy-dev-indexer"`,
  `Source="local"`, `Status=resolved`, `Reason=""`.
- *Base exists nowhere:* `Status=unknown`,
  `Reason="task/x is not a remote-tracking ref on origin and is not a local branch"`.
- *Git failure:* `Status=error`,
  `Reason="git rev-list failed (exit 128): Not a valid object name origin/x"`.

### 5.3 The two checks become honest and provenance-bearing (Q1 + Q2)

`CheckBranchMerged` / `CheckBranchNeedsRebase` (`pkg/wsm/git_utils.go`) take a
`gitclient.GitClient` (so existence checks share the backend and capture stderr)
and return a `BaseComparison`:

```go
func CheckBranchMerged(ctx, gc, repoPath, base, remote) (BaseComparison, error)
func CheckBranchNeedsRebase(ctx, gc, repoPath, base, remote) (BaseComparison, error)
```

Pseudocode for the merged check:

```text
func CheckBranchMerged(ctx, gc, repoPath, base, remote):
    res = ResolveBaseRef(gc, repoPath, base, remote)
    out = BaseComparison{ConfiguredBase: base, Remote: remote,
                         ResolvedRef: res.Ref, RefSource: res.Source,
                         Status: res.Status, Reason: res.Reason}
    if res.Status != resolved:
        return out           # IsMerged/NeedsRebase stay false; caller sees Status
    err = git merge-base --is-ancestor HEAD res.Ref   # via runGit → captures stderr
    if err:
        out.Status = error
        out.Reason = "merge-base failed: " + stderr(err)
        return out
    out.IsMerged = (err == nil)
    return out
```

The rebase check is symmetric, using `git rev-list --count HEAD..res.Ref` and
setting `NeedsRebase = (count != "0")`.

### 5.4 Default branch discovery (Q3)

Add a `DefaultBranch` method to the git client and persist the result.

```go
// pkg/wsm/gitclient/client.go (interface addition)
// DefaultBranch returns the remote's advertised default branch name
// (e.g. "main", "master", "develop"), without the "<remote>/" prefix.
// Returns "" if origin/HEAD is unset; caller may then probe candidates.
DefaultBranch(ctx, repo, remote) (string, error)
```

CLI impl (`cli_client.go`), reusing `runGit`:

```go
func (c *CliGitClient) DefaultBranch(ctx, repo, remote) (string, error) {
    if remote == "" { remote = "origin" }
    out, err := runGit(ctx, repo.Path(), "symbolic-ref",
        "refs/remotes/"+remote+"/HEAD")
    if err != nil { return "", nil }   // unset is not an error; probe below
    ref := strings.TrimSpace(string(out))             // refs/remotes/origin/develop
    prefix := "refs/remotes/" + remote + "/"
    return strings.TrimPrefix(ref, prefix), nil       // "develop"
}
```

Fallback in the resolver when `DefaultBranch` returns `""`: probe
`main`, `master`, `develop` via `RemoteTrackingBranchExists` and pick the first
present. (Document this order; it is a heuristic but covers the common cases.)

Persistence:

- Add `DefaultBaseBranch string` to `Repository` (`pkg/wsm/types.go:7-17`).
- Populate in `analyzeRepository` (`pkg/wsm/discovery.go:182`): after
  `RemoteURL`, call `gc.DefaultBranch(ctx, handle, "origin")`.
- Add to `RepositoryMetadata` (`workspace.go:1523-1528`) and emit it in
  `createWorkspaceMetadata` (`:1538-1547`) so `.wsm/wsm.json` records it.

### 5.5 Per-repo base override (Q4)

Add per-repo override fields to `RepositoryMetadata` (the in-workspace file):

```go
// pkg/wsm/workspace.go (RepositoryMetadata addition)
type RepositoryMetadata struct {
    Name         string   `json:"name"`
    Path         string   `json:"path"`
    Categories   []string `json:"categories"`
    WorktreePath string   `json:"worktreePath"`
    BaseBranch   string   `json:"baseBranch,omitempty"`  // NEW override
    BaseRemote   string   `json:"baseRemote,omitempty"`  // NEW override
}
```

`""` means "inherit the next precedence layer" (the common case), so existing
`wsm.json` files parse unchanged.

The **config-dir** store needs the same per-repo fields so `--global` has
somewhere to write. Add them to the `Repository` struct that the config-dir
JSON serializes:

```go
// pkg/wsm/types.go (Repository addition — config-dir store)
type Repository struct {
    ...
    DefaultBaseBranch string `json:"default_base_branch,omitempty"` // Q3 discovered
    BaseBranch        string `json:"base_branch,omitempty"`         // Q4 --global override
    BaseRemote        string `json:"base_remote,omitempty"`         // Q4 --global override
}
```

So a per-repo override can live in **either** store; the flag selects which.
Both default to `""` (inherit).

### 5.6 The new resolution precedence (Q3 + Q4)

`ResolveBaseBranch` becomes per-repo. Single source of truth:

```text
# pkg/wsm/branch — per-repo base resolution
func ResolveBaseBranchForRepo(repo, workspace) (base, remote):
    # 1) in-workspace per-repo override (Q4) — default `set-base` target
    if repo.BaseBranchWorkspace != "":           # from .wsm/wsm.json
        return repo.BaseBranchWorkspace, orDefault(repo.BaseRemoteWorkspace, "origin")
    # 2) config-dir per-repo override (Q4) — `set-base --global` target
    if repo.BaseBranchGlobal != "":              # from config-dir JSON
        return repo.BaseBranchGlobal, orDefault(repo.BaseRemoteGlobal, "origin")
    # 3) workspace-level base (existing Workspace.BaseBranch)
    if workspace.BaseBranch != "":
        return workspace.BaseBranch, "origin"
    # 4) discovered per-repo default (Q3)
    if repo.DefaultBaseBranch != "":
        return repo.DefaultBaseBranch, "origin"
    # 5) WSM_BASE_BRANCH env
    if v := os.Getenv("WSM_BASE_BRANCH"); v != "":
        return v, "origin"
    # 6) hardcoded fallback
    return "main", "origin"
```

Then `ResolveBaseRef` (§5.2) turns the chosen `base`/`remote` into a concrete
ref. The `BaseComparison.ConfiguredBase` field records *which* precedence layer
won, so the status table can show "develop (default)" vs "task/x (workspace)"
vs "task/y (global)" vs "main (fallback)". (In code the two override fields are
merged onto a single `Repository` during `LoadWorkspace`; the pseudocode names
them distinctly only to show provenance — see E3.)

### 5.7 New command: `wsm set-base` (Q4)

A thin Cobra command in `cmd/wsm/cmds/workspace/`:

```
wsm set-base <repo> [--workspace <ws>] --branch <branch> [--remote <remote>] [--fetch] [--global]
```

**Two targets, selected by a flag — never both at once (no mirroring):**

- **Default (no `--global`):** writes the per-repo `BaseBranch`/`BaseRemote`
  override to the **in-workspace `.wsm/wsm.json`** only. This is the
  workspace-specific store: it lives inside the workspace directory and travels
  with it. It does **not** touch the config-dir JSON.
- **`--global`:** writes the per-repo override to the **config-dir workspace
  JSON** (`~/.config/workspace-manager/workspaces/<name>.json`) only. This is
  the canonical/persistent store that survives workspace re-creation and is
  shared regardless of the in-workspace file.

Behavior (either mode):

1. Load the workspace (config-dir + in-workspace) and resolve `<repo>`.
2. Set the chosen store's per-repo `BaseBranch`/`BaseRemote` for `<repo>`.
3. If `--fetch`, run `git fetch <remote> <branch>` in the repo's worktree so the
   remote-tracking ref exists before the next `wsm status`.
4. Write **only** the targeted store (`.wsm/wsm.json` for default, config-dir
   JSON for `--global`). Do not mirror.
5. Print the resolved precedence for confirmation, naming the store written
   (e.g. `goldeneaglecoin.com base → develop (origin) [stored: workspace]`).

Because in-workspace overrides beat config-dir overrides at load time (E3), a
local `set-base` silently supersedes an earlier `--global` for the same repo —
this is intended (local beats global) and the printout in step 5 makes it
visible.

### 5.8 Status table changes (Q1 + Q2)

`printStatusDetailed` (`cmd/wsm/cmds/workspace/status.go`) gains a `BASE`
column and makes `MERGED`/`REBASE` honest:

```
REPOSITORY  BRANCH               BASE                             STATUS   ...  MERGED  REBASE
geppetto    task/ragkit-coinv..  task/deploy-dev-indexer (local)   clean    ...  -       ✓
goldeneag.. main                 origin/develop                    clean    ...  ✓       -
flowkit     task/ragkit-coinv..  ? no base ref (origin+local)       clean    ...  ?       ?
```

`getMergedString`/`getRebaseString` switch on `Base.Status`:

```text
func getMergedString(s RepositoryStatus) string:
    switch s.Base.Status:
    case resolved: return s.Base.IsMerged ? "✓" : "-"
    case unknown:  return "?"                      # honest "couldn't tell"
    case error:    return "!"                      # something broke
```

`statusToRows` (JSON output) gains `base`, `base_ref`, `base_status`,
`base_reason` columns (additive; existing `is_merged`/`needs_rebase` stay).

## 6. Decision Records

### Decision: E1 — One `BaseComparison` struct, not separate "reason" fields

- **Context:** Q1 (reason) and Q2 (which branch) could be implemented as
  ad-hoc extra `string` fields. But they are facets of one outcome.
- **Options considered:** (1) add `BaseRef string` + `BaseReason string`
  separately; (2) bundle into `BaseComparison`.
- **Decision:** Bundle into `BaseComparison`. Keep `IsMerged`/`NeedsRebase` as
  top-level bool mirrors for JSON compat.
- **Rationale:** A consumer reading JSON gets one coherent object; the table
  renderer reads one struct; the checks return one type. Avoids the "set reason
  but forgot to set ref" class of bugs.
- **Consequences:** `statusToRows` and `printStatusDetailed` both change, but in
  one place each. New fields are additive.
- **Status:** proposed

### Decision: E2 — Three status classes: resolved / unknown / error

- **Context:** "Couldn't compare" has two distinct causes: no ref exists
  (unknown, benign) vs git broke (error, actionable). Conflating them hides
  real breakage.
- **Options considered:** (1) just "known/unknown"; (2) resolved/unknown/error.
- **Decision:** Three classes. `unknown` = no ref (forked workspace,
  misconfigured base). `error` = git failure (corrupt repo, disk issue).
- **Rationale:** `unknown` is a configuration fact; `error` is a bug. The user
  fixes them differently (set a base vs investigate git).
- **Consequences:** Two glyphs in the table (`?` and `!`). Slightly more logic
  in the renderers.
- **Status:** proposed

### Decision: E3 — Two override stores, flag-selected, in-workspace beats config-dir at load time

- **Context:** `wsm status` loads from config-dir JSON
  (`WorkspaceContextService.LoadWorkspace` → `LoadWorkspaces`,
  `workspace_context.go:21`), but the file users see and edit is the in-workspace
  `.wsm/wsm.json`. A per-repo override needs to be expressible in both, and the
  user asked that `set-base` default to the workspace-specific store with
  `--global` for the config store.
- **Options considered:** (1) one store only (config-dir); (2) one store only
  (in-workspace, `LoadWorkspace` reads it directly); (3) two stores, `set-base`
  mirrors to both; (4) two stores, flag-selected, no mirroring, in-workspace
  wins on conflict.
- **Decision:** Option (4). `set-base` (default) writes only `.wsm/wsm.json`;
  `set-base --global` writes only the config-dir workspace JSON. `LoadWorkspace`
  reads the config-dir `Workspace` (with its per-repo `Repository.BaseBranch`),
  then overlays in-workspace `.wsm/wsm.json` per-repo overrides on top, so an
  in-workspace override supersedes a config-dir one for the same repo.
- **Rationale:** Mirroring (option 3) creates two sources of truth that silently
  drift; flag-selection (option 4) keeps each write to one store and makes the
  precedence explicit and auditable (the status `BASE` column names the winning
  layer). Local-beats-global is the least surprising override rule, and the
  default target is the most local, highest-precedence store — so the common
  case "set a base for this worktree" just works without a flag.
- **Consequences:** Both `Repository` (config-dir, `types.go:7-17`) and
  `RepositoryMetadata` (in-workspace, `workspace.go:1522-1528`) gain
  `BaseBranch`/`BaseRemote`. `LoadWorkspace` (`workspace.go:516`) gains an overlay
  merge step (in-workspace onto config-dir). Tests must cover: default writes
  only `.wsm/wsm.json`; `--global` writes only config-dir; in-workspace wins on
  conflict; absence of both falls through to `Workspace.BaseBranch`.
- **Status:** proposed

### Decision: E4 — Detect default branch via `symbolic-ref refs/remotes/<r>/HEAD`, probe candidates on miss

- **Context:** Hardcoding `main` is wrong for `develop`/`master` repos. Need a
  remote-advertised default that needs no configuration.
- **Options considered:** (1) always `main`; (2) probe `main`/`master`/`develop`
  in order; (3) `symbolic-ref refs/remotes/origin/HEAD` first, probe on miss.
- **Decision:** Option (3). `symbolic-ref` is the remote's own advertised
  default (most correct); probing covers the rare unset case.
- **Rationale:** Verified `symbolic-ref` returns `develop` for
  `goldeneaglecoin.com` and `main` for `geppetto` — exactly the right answer in
  both. Probing is a documented heuristic fallback.
- **Consequences:** New `GitClient.DefaultBranch` method + impl. Probe order
  (`main`, `master`, `develop`) must be documented and tested.
- **Status:** proposed

### Decision: E5 — Override precedence: per-repo > workspace > discovered default > env > main

- **Context:** Multiple sources can name a base; need a deterministic order.
- **Options considered:** (1) workspace always wins; (2) most-specific-first.
- **Decision:** Most-specific-first (per-repo override, then workspace, then
  discovered default, then env, then `main`). `ConfiguredBase` records which
  layer won.
- **Rationale:** A per-repo override only exists because the user explicitly
  asked for it; it must beat a global default. The discovered default (Q3) beats
  the hardcoded `main` because it reflects the repo's actual convention.
- **Consequences:** `ResolveBaseBranchForRepo` centralizes the order; the status
  table's `BASE` column shows the source so the precedence is auditable.
- **Status:** proposed

## 7. Pseudocode and Key Flows

### 7.1 End-to-end status with all four enhancements

```text
wsm status <ws>:
    workspace = LoadWorkspace(ws)                 # config-dir + overlay in-workspace overrides (E3: local wins)
    for each repo (concurrent via errgroup):
        base, remote = ResolveBaseBranchForRepo(repo, workspace)   # E5 precedence
        cmp = CheckBranchMerged(ctx, gc, repoPath, base, remote)    # returns BaseComparison
        cmpNeeds = CheckBranchNeedsRebase(ctx, gc, repoPath, base, remote)
        # merge into one BaseComparison (same base/ref; IsMerged from merged, NeedsRebase from rebase)
        status.Base = cmp   # with NeedsRebase filled from cmpNeeds
        status.IsMerged = cmp.IsMerged        # mirror for compat
        status.NeedsRebase = cmpNeeds.NeedsRebase
    render: BASE column shows ResolvedRef + "(" + RefSource + ")" or "?" + Reason
            MERGED/REBASE columns honor Base.Status (✓/-/?/!)
```

### 7.2 Discovery recording the default branch (Q3)

```text
analyzeRepository(path):
    repo = &Repository{Name: base(path), Path: path, ...}
    h = gc.Open(path)
    repo.RemoteURL = gc.RemoteURL(h, "origin")
    repo.CurrentBranch = gc.CurrentBranch(h)
    repo.DefaultBaseBranch = gc.DefaultBranch(h, "origin")   # NEW (E4): "develop" / "main" / ""
    if repo.DefaultBaseBranch == "":
        for cand in ["main","master","develop"]:
            if gc.RemoteTrackingBranchExists(h, "origin", cand):
                repo.DefaultBaseBranch = cand; break
    repo.Branches = gc.ListLocalBranches(h)
    ...
    return repo
```

### 7.3 `set-base` flow (Q4)

```text
wsm set-base goldeneaglecoin.com --branch develop --fetch:
    ws = LoadWorkspace(detect or --workspace)
    # default mode → in-workspace store only
    meta = read .wsm/wsm.json
    find repo "goldeneaglecoin.com" in meta.Repositories
    repo.BaseBranch = "develop"; repo.BaseRemote = "" (default origin)
    if --fetch: git -C <worktreePath> fetch origin develop
    write .wsm/wsm.json          # ONLY this file; do NOT touch config-dir
    print "goldeneaglecoin.com base → develop (origin) [stored: workspace]"

wsm set-base goldeneaglecoin.com --branch develop --global:
    ws = LoadWorkspace(detect or --workspace)        # config-dir Workspace
    find repo "goldeneaglecoin.com" in ws.Repositories
    repo.BaseBranch = "develop"; repo.BaseRemote = ""
    if --fetch: git -C <worktreePath> fetch origin develop
    SaveWorkspace(ws)           # ONLY config-dir JSON; do NOT touch .wsm/wsm.json
    print "goldeneaglecoin.com base → develop (origin) [stored: global]"
```

At status time, `LoadWorkspace` overlays in-workspace overrides onto config-dir
ones, so if both exist for the same repo the in-workspace value wins.

### 7.4 Precedence + resolution diagram

```text
   .wsm/wsm.json (in-workspace)        config-dir JSON                 Workspace.BaseBranch
     BaseBranch / BaseRemote             BaseBranch / BaseRemote           (workspace base)
     default set-base target             --global set-base target
            │                                     │                            │
            ▼   E3 overlay merge (in-workspace wins on conflict) ──────────────┘
              ResolveBaseBranchForRepo(repo, ws)
                (in-workspace per-repo > config-dir per-repo > workspace base
                 > discovered default > env > main)
                        │
                        ▼
                 chosen base, remote
                        │
                        ▼
              ResolveBaseRef(base, remote):  prefer <remote>/<base> → local <base> → unknown
                        │
                        ▼
            BaseRefResolution {Ref, Source, Status, Reason}
                        │
                        ▼
            CheckBranchMerged / CheckBranchNeedsRebase  (runGit, capture stderr)
                        │
                        ▼
            BaseComparison {ConfiguredBase, ResolvedRef, RefSource, Status, Reason, IsMerged, NeedsRebase}
                        │
                        ▼
            RepositoryStatus.Base  →  status table BASE / MERGED / REBASE columns
```

## 8. Implementation Phases

Each phase is independently testable and committable. Phases reuse the base
fix's `ResolveBaseRef` (from `01-...`) and extend it.

### Phase E1 — `BaseComparison` model + honest checks (Q1 + Q2 core)

- `pkg/wsm/types.go`: add `BaseComparisonStatus`, `BaseComparison`; add
  `Base BaseComparison` to `RepositoryStatus`; keep `IsMerged`/`NeedsRebase`.
- `pkg/wsm/branch/status_resolve.go` (new): `BaseRefResolution` +
  `ResolveBaseRef` returning the structured outcome (prefer remote-tracking,
  fall back local, else `unknown` with reason; `error` with captured stderr on
  git failure).
- `pkg/wsm/git_utils.go`: rewrite `CheckBranchMerged`/`CheckBranchNeedsRebase`
  to take a `gitclient.GitClient`, use `ResolveBaseRef`, run via `runGit`
  (capture stderr), and return `BaseComparison`.
- `pkg/wsm/status.go:151-157`: call the new checks; set `status.Base` and the
  bool mirrors.

### Phase E2 — Default branch discovery + persistence (Q3)

- `pkg/wsm/gitclient/client.go`: add `DefaultBranch` to the interface.
- `pkg/wsm/gitclient/cli_client.go`: implement via `symbolic-ref`, with the
  candidate-probe fallback.
- `pkg/wsm/types.go`: add `DefaultBaseBranch string` to `Repository`.
- `pkg/wsm/discovery.go:182`: populate it in `analyzeRepository`.
- `pkg/wsm/workspace.go`: add to `RepositoryMetadata`; emit in
  `createWorkspaceMetadata`.
- Test: a fixture with `origin/HEAD -> develop` → `DefaultBaseBranch ==
  "develop"`; a fixture with unset HEAD and only `origin/master` → fallback
  returns `"master"`.

### Phase E3 — Per-repo override storage + precedence (Q4 core)

- `pkg/wsm/workspace.go`: add `BaseBranch`/`BaseRemote` to `RepositoryMetadata`.
- `pkg/wsm/workspace.go:516` (`LoadWorkspace`): after loading the config-dir
  `Workspace`, read the in-workspace `.wsm/wsm.json` and merge per-repo
  `BaseBranch`/`BaseRemote` onto the matching `Repository` entries (E3).
- `pkg/wsm/branch`: add `ResolveBaseBranchForRepo(repo, ws)` implementing the E5
  precedence; have the status path use it instead of the single
  `ResolveBaseBranch`.
- Test: override in `.wsm/wsm.json` beats workspace base and discovered default;
  absence falls through to discovered default; absence of both falls to `main`.

### Phase E4 — `wsm set-base` command (Q4 surface)

- `cmd/wsm/cmds/workspace/set_base.go` (new): Cobra command; flags `--branch`,
  `--remote`, `--workspace`, `--fetch`, `--global`.
- Workflow:
  - **Default:** load workspace, edit `.wsm/wsm.json` per-repo override,
    `--fetch` if requested, write `.wsm/wsm.json` only, print precedence
    (`[stored: workspace]`).
  - **`--global`:** load workspace, edit config-dir `Repository` per-repo
    override, `--fetch` if requested, `SaveWorkspace` (config-dir only), print
    precedence (`[stored: global]`).
  - Never write both stores in one invocation.
- Wire into the workspace command group.
- Test: `set-base goldeneaglecoin.com --branch develop` then `wsm status` shows
  `origin/develop` for that repo and `.wsm/wsm.json` is the only file changed;
  `set-base ... --global` changes only the config-dir JSON; an in-workspace
  override supersedes a config-dir one for the same repo.

### Phase E5 — Status table + JSON columns (Q1 + Q2 surface)

- `cmd/wsm/cmds/workspace/status.go`:
  - `statusToRows`: add `base`, `base_ref`, `base_status`, `base_reason` rows.
  - `printStatusDetailed`: add `BASE` column; render `MERGED`/`REBASE` via
    `Base.Status` (`✓`/`-`/`?`/`!`).
  - `getMergedString`/`getRebaseString`: switch on `Base.Status`.
- Test: golden output for resolved/unknown/error rows.

### Phase E6 — Validate end-to-end on the real workspace

- `wsm status` on `ragkit-coinvault-mysql`: confirm
  `geppetto` shows `task/deploy-dev-indexer (local)` and real booleans;
  `goldeneaglecoin.com` shows `origin/develop` (Q3); no `exit status 128` logs.
- `wsm set-base goldeneaglecoin.com --branch develop --fetch`; re-run status;
  confirm override column and that fetch materialized the ref.

## 9. Test Strategy

- **Unit (resolver):** `ResolveBaseRef` for (remote-tracking, local fallback,
  neither) → `resolved`/`unknown` with correct `Reason`; injected git failure →
  `error` with stderr.
- **Unit (discovery):** `DefaultBranch` via `symbolic-ref` (develop, main); unset
  → candidate probe returns `master`/`develop`/`main` as available; none → `""`.
- **Unit (precedence):** `ResolveBaseBranchForRepo` table-driven across the five
  layers; assert which `ConfiguredBase` wins.
- **Unit (merge step):** `LoadWorkspace` merges `.wsm/wsm.json` per-repo override
  over config-dir; absent override leaves config-dir value untouched.
- **Integration (set-base):** write override, fetch, re-status; assert the repo's
  `Base.ResolvedRef` changed and others did not.
- **Regression:** `TestStatus_MixedWorkspace_DifferentDefaults` — a workspace
  with one `develop`-default repo and one `main`-default repo reports each
  against its own default without explicit overrides (Q3).
- **Regression:** `TestStatus_PerRepoOverride` — a `.wsm/wsm.json` override makes
  status compare one repo against a different branch than the rest (Q4).
- **Manual:** the two commands in Phase E6 on the real workspace.

Run:

```bash
go test ./pkg/wsm/... -count=1 -run 'Status|Branch|Rebase|Merged|BaseRef|DefaultBranch|SetBase'
go test ./... -count=1
golangci-lint run ./pkg/wsm/... ./cmd/wsm/cmds/workspace/...
```

## 10. Risks, Alternatives, Open Questions

### Risks

- **Two-store drift (E3):** The two stores are no longer mirrored, so a user
  can set a `--global` override and later a local one for the same repo without
  the config-dir value being overwritten. This is intended (local beats global),
  but means the config-dir value can look "stale" next to the active local one.
  Mitigation: `LoadWorkspace` overlays in-workspace onto config-dir at read time
  and the status `BASE` column names the winning layer, so the effective value is
  always visible; `set-base` prints `[stored: workspace|global]` so the writer
  knows which store they touched. No automatic syncing is attempted.
- **Default branch changes upstream:** If a repo's remote re-points
  `origin/HEAD` (e.g. `develop` → `main`), the cached `DefaultBaseBranch` goes
  stale. Mitigation: re-discovery (`wsm discover`) refreshes it; document that
  `set-base` overrides take precedence anyway.
- **Probe order bias:** Falling back to `main`/`master`/`develop` could pick
  `main` for a repo whose real default is `develop` but whose HEAD is unset.
  Low risk (`symbolic-ref` is almost always set); documented as a heuristic.
- **Extra git calls:** `DefaultBranch` (one `symbolic-ref`) + two
  `for-each-ref` for `ResolveBaseRef`, per repo. All local, read-only, cheap;
  safe under the existing `errgroup` concurrency.

### Alternatives considered

- **Store base ref SHA at fork time.** More robust to ref movement, but a bigger
  format change and a read-only status command shouldn't depend on fork-time
  state. Defer.
- **Make `IsMerged`/`NeedsRebase` nullable `*bool`.** Cleanest semantically but
  breaks JSON consumers and the table helpers. Rejected in favor of additive
  `Base` struct (E1).
- **Auto-fetch the base in status.** Mutates remote-tracking refs as a side
  effect of a read-only command and may 404 for local-only bases. Rejected;
  `--fetch` already exists for opt-in; `set-base --fetch` is the targeted form.

### Open questions

- Should the `BASE` column show the precedence source (e.g.
  `develop (default)`, `task/x (override)`) by default, or only with `--verbose`?
  Lean: show source always; it's the whole point of Q2.
- Should `wsm set-base` support `--clear` to remove an override (revert to
  inherited/workspace/discovered)? Likely yes — trivial extension.
- Should discovery re-detect `DefaultBaseBranch` on every `wsm status`, or only
  on explicit `wsm discover`? Lean: only on `discover` (cheap to store, avoids
  surprise git calls on every status).

## 11. References (key files)

| File | Why it matters |
| --- | --- |
| `pkg/wsm/types.go:46-58` | `RepositoryStatus` — add `Base BaseComparison`; keep bool mirrors |
| `pkg/wsm/types.go:7-17` | `Repository` — add `DefaultBaseBranch` |
| `pkg/wsm/git_utils.go:24-101` | `CheckBranchMerged`/`CheckBranchNeedsRebase` — rewrite to return `BaseComparison` |
| `pkg/wsm/status.go:119-160` | `getRepositoryStatusWithClient` — call new checks, set `status.Base` |
| `pkg/wsm/branch/types.go:19-36` | `ResolveBaseBranch`/`DefaultBaseBranch` — extend to per-repo precedence |
| `pkg/wsm/branch/status_resolve.go` (new) | `BaseRefResolution` + `ResolveBaseRef` |
| `pkg/wsm/gitclient/client.go` | `GitClient` interface — add `DefaultBranch` |
| `pkg/wsm/gitclient/cli_client.go:30-38,107-126` | `runGit` (captures stderr) + existence-check impls to reuse |
| `pkg/wsm/discovery.go:182-201` | `analyzeRepository` — populate `DefaultBaseBranch` |
| `pkg/wsm/workspace.go:1510-1547` | `WorkspaceMetadata`/`RepositoryMetadata`/`createWorkspaceMetadata` — add override + default fields |
| `pkg/wsm/workspace.go:449-528` | `LoadWorkspaces`/`LoadWorkspace` — merge `.wsm/wsm.json` overrides (E3) |
| `pkg/wsm/workspace_context.go:21-35` | `WorkspaceContextService.LoadWorkspace` — the path `wsm status` actually uses |
| `pkg/wsm/workflows/status_workflow.go` | `StatusWorkflow.GetStatus` — entry from CLI |
| `cmd/wsm/cmds/workspace/status.go` | table + JSON rendering — add `BASE` column, honest `MERGED`/`REBASE` |
| `cmd/wsm/cmds/workspace/set_base.go` (new) | `wsm set-base` command (Q4) |
| `pkg/wsm/workflows/fork_workflow.go:52-88` | `ForkWorkflow.Plan` — context for why base is often local-only |
| `/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql/.wsm/wsm.json` | failing workspace manifest |
| `/home/manuel/workspaces/2026-08-13/ragkit-coinvault-mysql/goldeneaglecoin.com` | repo with `origin/HEAD -> develop` (Q3 evidence) |

### Glossary

- **Base branch:** the branch a workspace/fork compares against; stored as
  `Workspace.BaseBranch` (workspace-level) and, new, per-repo in
  `RepositoryMetadata.BaseBranch`.
- **Remote-tracking ref:** local cache of a remote branch, e.g.
  `origin/develop` (`refs/remotes/origin/develop`).
- **Default base branch (Q3):** the remote's advertised default, from
  `git symbolic-ref refs/remotes/origin/HEAD`; stored per repo as
  `Repository.DefaultBaseBranch`.
- **Per-repo override (Q4):** a `BaseBranch`/`BaseRemote` set on one repo in
  `.wsm/wsm.json` that beats the workspace-level base.
- **`BaseComparison`:** the provenance-bearing result of a merge/rebase check
  (ref used, source, status, reason, values).
- **resolved / unknown / error:** the three comparison outcomes; `unknown` = no
  ref to compare (config), `error` = git failed (bug).
