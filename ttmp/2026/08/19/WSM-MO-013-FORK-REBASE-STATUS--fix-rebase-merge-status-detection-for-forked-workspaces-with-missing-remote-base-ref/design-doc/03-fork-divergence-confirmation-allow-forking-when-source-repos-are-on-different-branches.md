---
Title: Fork Divergence Confirmation — Allow Forking When Source Repos Are on Different Branches
Ticket: WSM-MO-013-FORK-REBASE-STATUS
Status: active
Topics:
    - workspace-manager
    - git
    - fork
    - bugfix
    - cli
    - api-design
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/wsm/workflows/fork_workflow.go
      Note: Plan hard-fails on branch divergence (line 78-84)
ExternalSources: []
Summary: 'Intern-oriented analysis, design, and implementation guide for the fork-divergence enhancement: when a source workspace''s repositories are on different branches, wsm fork should prompt for confirmation (or accept a flag) and allow choosing a different default upstream base branch instead of hard-failing.'
LastUpdated: 2026-08-19T11:30:00-04:00
WhatFor: Onboarding an engineer to the fork-divergence fix and the interactive confirmation conventions used across wsm commands.
WhenToUse: Read this before changing pkg/wsm/workflows/fork_workflow.go or cmd/wsm/cmds/workspace/fork.go.
---


# Fork Divergence Confirmation — Allow Forking When Source Repos Are on Different Branches

> Audience: a new engineer (intern-level). Assumes you have read design-doc 01
> (the status bug) at least for the layering model. This guide is self-contained
> for the fork-divergence change.

## 1. Executive Summary

`wsm fork <new-name> <source>` currently derives the fork's base branch from
the first source repository's current branch and **requires every source repo
to be on the same branch**, hard-failing otherwise:

```
$ wsm fork ttc-admin-chat deploy-dev-indexer
Error: repositories in source workspace are on different branches:
       goldeneaglecoin.com is on task/deploy-image, but expected task/deploy-dev-indexer
```

This blocks a legitimate workflow: forking a workspace whose repos have drifted
onto different task branches (common when one repo got a side-fix on its own
branch). The fix lets the fork proceed by **confirming which branch to use as
the base**, either interactively (a `huh` prompt, matching the existing
delete/create confirmation pattern) or via a new `--base-branch` flag for
non-interactive/automation use.

The change is split across two layers, following the WSM layering rule (CLI
adapters own interaction; workflows stay non-interactive):

1. **Workflow** (`pkg/wsm/workflows/fork_workflow.go`): stop hard-failing on
   divergence; instead accept an explicit `BaseBranch` on `ForkRequest` and
   return a typed `ErrBranchDivergence` carrying the per-repo branch map so the
   caller can prompt.
2. **CLI** (`cmd/wsm/cmds/workspace/fork.go`): on divergence, prompt with
   `huh` (select base from observed branches + confirm), then re-run with the
   chosen branch. In glaze/non-interactive mode, require `--base-branch`.

## 2. Problem Statement and Scope

### 2.1 The user-reported symptom

```
wsm fork ttc-admin-chat deploy-dev-indexer
Error: repositories in source workspace are on different branches:
       goldeneaglecoin.com is on task/deploy-image, but expected task/deploy-dev-indexer
```

The source workspace `deploy-dev-indexer` has most repos on
`task/deploy-dev-indexer` but `goldeneaglecoin.com` on `task/deploy-image`.

### 2.2 Desired behavior

- Ask for confirmation to use the current branch when it diverges from the name
  of the workspace (the conventional branch is `task/<workspace-name>`, built by
  `BuildWorkspaceBranch`).
- Allow the user to pick a **different default upstream branch** as the fork's
  base, so the fork still succeeds.

### 2.3 In scope

- `ForkWorkflow.Plan` divergence handling + `ForkRequest.BaseBranch`.
- A typed `ErrBranchDivergence`.
- The `--base-branch` CLI flag.
- The interactive `huh` select+confirm in `fork.go`.
- Non-interactive (glaze-output) fallback requiring `--base-branch`.

### 2.4 Out of scope

- The status `BaseComparison` work (design-doc 02, phases E1–E6).
- Changing `BuildWorkspaceBranch` conventions.
- Per-repo base overrides at fork time (a repo may still be on a divergent
  branch; the fork uses one base for all — consistent with today's model).

## 3. Current-State Analysis (evidence)

### 3.1 The hard-fail

```go
// pkg/wsm/workflows/fork_workflow.go:78-84
baseBranch := status.Repositories[0].CurrentBranch
if baseBranch == "" {
    return nil, errors.New("failed to detect base branch from source workspace")
}
for _, repoStatus := range status.Repositories {
    if repoStatus.CurrentBranch != baseBranch {
        return nil, errors.Errorf("repositories in source workspace are on different branches: %s is on %s, but expected %s",
            repoStatus.Repository.Name, repoStatus.CurrentBranch, baseBranch)
    }
}
```

So the base is `Repositories[0].CurrentBranch` and uniformity is mandatory.

### 3.2 The branch-name convention

```go
// pkg/wsm/workflows/create_workflow.go:64-72
func BuildWorkspaceBranch(workspaceName, branch, branchPrefix string) (string, bool) {
    if branch != "" { return branch, false }
    prefix := branchPrefix
    if prefix == "" { prefix = "task" }
    return fmt.Sprintf("%s/%s", prefix, workspaceName), true
}
```

The conventional branch for a workspace named `deploy-dev-indexer` is
`task/deploy-dev-indexer`. This is the "expected" branch the user refers to when
they say "diverges from the name of the workspace".

### 3.3 The existing confirmation pattern (what we must mirror)

WSM already has interactive confirmation via `charmbracelet/huh`, with a clean
cancel convention and an `allowPrompt` gate for non-interactive mode.

```go
// cmd/wsm/cmds/workspace/helpers.go:13-22
func isUserCancelledError(err error) bool {
    errMsg := strings.ToLower(err.Error())
    return strings.Contains(errMsg, "cancelled") ||
        strings.Contains(errMsg, "aborted") ||
        strings.Contains(errMsg, "interrupt") // (and a few more)
}
```

```go
// cmd/wsm/cmds/workspace/delete.go:170-189  (the canonical confirm gate)
if !settings_.Force {
    if !allowPrompt {
        return nil, errors.New("--force is required when using --with-glaze-output")
    }
    var confirmed bool
    form := huh.NewForm(huh.NewGroup(
        huh.NewConfirm().
            Title(fmt.Sprintf("Are you sure you want to delete workspace '%s'?", workspaceName)).
            Description("This action cannot be undone.").
            Value(&confirmed),
    ))
    if err := form.Run(); err != nil {
        if isUserCancelledError(err) { return &deleteExecutionResult{Cancelled: true}, nil }
        return nil, errors.Wrap(err, "confirmation failed")
    }
    if !confirmed { return &deleteExecutionResult{Cancelled: true}, nil }
}
```

The two execution entrypoints set the gates:

```go
// delete.go:72   Run()                  -> execute(ctx, vals, true,  true)   // human + prompt
// delete.go:98   RunIntoGlazeProcessor()-> execute(ctx, vals, false, false)  // JSON, no prompt
```

So `Run` (human TTY) allows prompts; `RunIntoGlazeProcessor` (glaze/JSON) does
not, and requires a flag instead. Fork must follow the same shape.

### 3.4 Fork already supports cancellation

```go
// cmd/wsm/cmds/workspace/fork.go:182-187
workspace, _, err := workflow.Fork(ctx, req)
if err != nil {
    if isUserCancelledError(err) { return &forkExecutionResult{Cancelled: true}, nil }
    return nil, err
}
```

`forkExecutionResult.Cancelled` already exists and is honored in `Run`/`RunIntoGlazeProcessor`. So we have a place to land a user-cancelled divergence prompt.

## 4. Gap Analysis

- **G1:** `Plan` hard-fails on divergence (`fork_workflow.go:80`); there is no
  way to proceed.
- **G2:** `ForkRequest` has no explicit base-branch field (`Branch` is the
  *new* workspace's branch, not the base). So even if the CLI wanted to pass a
  chosen base, it cannot.
- **G3:** The divergence error is a plain `errors.Errorf` string — the CLI
  cannot programmatically inspect it to build a prompt (which branches? which
  repos?). A typed error is needed.
- **G4:** Fork's `execute` does not thread `emitHuman`/`allowPrompt` like
  delete's does, so it cannot distinguish interactive vs glaze mode for the
  prompt gate.

## 5. Proposed Architecture and APIs

### 5.1 `ForkRequest` gains an explicit base branch

```go
// pkg/wsm/workflows/fork_workflow.go
type ForkRequest struct {
    NewWorkspaceName    string
    SourceWorkspaceName string
    Branch              string // new workspace's branch (existing)
    BranchPrefix        string
    BaseBranch          string // NEW: explicit base/upstream branch override
    AgentSource         string
    DryRun              bool
}
```

`BaseBranch` is the branch the fork is created from (the "default upstream").
When set, `Plan` uses it directly and skips the uniformity check.

### 5.2 A typed divergence error

```go
// pkg/wsm/workflows/fork_workflow.go
// ErrBranchDivergence is returned by Plan when source repos are on different
// branches and no explicit BaseBranch was provided. It carries the per-repo
// branch map and the conventional "expected" branch so the caller can prompt.
type ErrBranchDivergence struct {
    Branches map[string]string // repo name -> current branch
    Expected string           // conventional task/<source-workspace-name>
    Source   string           // source workspace name (for messaging)
}
func (e *ErrBranchDivergence) Error() string { ... }
func (e *ErrBranchDivergence) DistinctBranches() []string { ... } // sorted unique branches
```

### 5.3 `Plan` resolution logic

```text
func Plan(ctx, req) (*ForkPlan, error):
    ...load source workspace + status...
    # 1) explicit override (from --base-branch) wins; no uniformity check
    if req.BaseBranch != "":
        baseBranch = req.BaseBranch
    else:
        # 2) detect from repos
        branches = { repo.CurrentBranch for repo in status.Repositories }
        distinct = unique(branches)
        if len(distinct) == 1:
            baseBranch = distinct[0]
        else:
            # 3) divergent -> typed error; caller prompts or passes --base-branch
            return nil, &ErrBranchDivergence{
                Branches: branches,
                Expected: BuildWorkspaceBranch(sourceWorkspaceName, "", "task"),
                Source:   sourceWorkspaceName,
            }
    # build plan with baseBranch (existing rest unchanged)
    return &ForkPlan{BaseBranch: baseBranch, ...}, nil
```

Note: the conventional `Expected` uses the **source** workspace name
(`BuildWorkspaceBranch(sourceWorkspaceName, "", "task")`), since the divergence
is relative to the source's convention.

### 5.4 CLI: `--base-branch` flag + interactive prompt

New flag on `ForkSettings`/command:

```
fields.New("base-branch", fields.TypeString, fields.WithHelp("Base/upstream branch to fork from (use when source repos are on different branches)")),
```

`execute` threads `emitHuman`/`allowPrompt` (matching delete) and handles
divergence:

```text
func execute(ctx, vals, emitHuman, allowPrompt) (*forkExecutionResult, error):
    ...build req...
    req.BaseBranch = settings_.BaseBranch
    plan, err := workflow.Plan(ctx, req)
    if err != nil:
        var div *ErrBranchDivergence
        if errors.As(err, &div):
            # non-interactive without flag -> require the flag
            if !allowPrompt:
                return nil, errors.Errorf(
                    "source repos are on different branches; "+
                    "pass --base-branch (branches: %s)", join(div.DistinctBranches()))
            # interactive: pick a base, then confirm
            chosen, ok, cancelled = promptBaseBranch(div)
            if cancelled: return &forkExecutionResult{Cancelled: true}, nil
            if !ok: return nil, errors.New("no base branch selected")
            req.BaseBranch = chosen
            plan, err = workflow.Plan(ctx, req)   # re-plan with explicit base
            if err != nil: return nil, err
        else:
            return nil, err
    ...workflow.Fork(ctx, req)...
```

### 5.5 The interactive prompt (select + confirm)

```text
func promptBaseBranch(div *ErrBranchDivergence) (chosen string, ok, cancelled bool):
    # default to the most common branch, then the expected conventional branch
    options = div.DistinctBranches()             # sorted unique observed
    default  = mostFrequent(div.Branches)
    if div.Expected not in options: options = append([div.Expected], options...)

    var branch string = default
    var confirm bool
    form = huh.NewForm(huh.NewGroup(
        huh.NewSelect[string]().
            Title("Source repos are on different branches. Choose the base branch to fork from:").
            Options(huh.NewOptions(options...)).
            Value(&branch),
        huh.NewConfirm().
            Title(fmt.Sprintf("Fork using base branch '%s'?", branch)).
            Description(showDivergence(div)).
            Value(&confirm),
    ))
    if err := form.Run(); err != nil:
        if isUserCancelledError(err): return "", false, true
        return "", false, false   // propagate as error upstream
    return branch, confirm, false
```

`showDivergence` renders the per-repo branch map so the user sees exactly which
repo is on which branch before confirming.

## 6. Decision Records

### Decision: F1 — Workflow returns a typed divergence error; CLI owns the prompt

- **Context:** The divergence is detected in the workflow, but interaction
  belongs in the CLI layer (layering rule; delete/create already do this).
- **Options considered:** (1) prompt inside the workflow; (2) workflow returns
  a typed error, CLI prompts; (3) workflow returns a `Plan` with a divergence
  flag and no error.
- **Decision:** Option (2). `Plan` returns `*ErrBranchDivergence`; the CLI does
  `errors.As` and prompts, then re-calls `Plan` with `req.BaseBranch` set.
- **Rationale:** Keeps the workflow pure/testable (no `huh` import in `pkg/wsm`)
  and matches the delete workflow pattern. A typed error carries the data the
  CLI needs (branch map) without changing `Plan`'s signature.
- **Consequences:** `fork_workflow.go` defines `ErrBranchDivergence`; `fork.go`
  gains a `errors.As` branch and a re-plan. Tests can assert the typed error
  directly without a TTY.
- **Status:** proposed

### Decision: F2 — `--base-branch` flag is the non-interactive escape hatch

- **Context:** Glaze/JSON mode (`RunIntoGlazeProcessor`) cannot show a `huh`
  prompt; CI/automation needs a way to fork a divergent workspace.
- **Options considered:** (1) auto-pick the majority branch silently; (2) require
  `--base-branch`; (3) add `--allow-diverged`.
- **Decision:** Option (2). When `!allowPrompt` and divergence is detected,
  require `--base-branch` and error with the observed branches listed.
- **Rationale:** Silent auto-picking hides a meaningful choice (which base?);
  an explicit flag is auditable and scriptable. Mirrors delete's
  `--force is required when using --with-glaze-output`.
- **Consequences:** One new flag. Scripts that fork divergent workspaces must
  pass `--base-branch`.
- **Status:** proposed

### Decision: F3 — Prompt defaults to the most frequent observed branch, offers all + the conventional one

- **Context:** The user asked to "use the current branch" but also "a different
  default upstream branch" — implying a choice, not just a confirm.
- **Options considered:** (1) confirm-only on the first repo's branch; (2) select
  among observed branches + confirm; (3) free-text entry.
- **Decision:** Option (2). A `huh.NewSelect` of distinct observed branches
  (default = most frequent), plus the conventional `task/<source-name>` if
  absent, followed by a `huh.NewConfirm`.
- **Rationale:** Most-frequent is the least surprising default (usually the
  majority branch matches the workspace's intent); offering the conventional
  branch covers the "diverges from the name of the workspace" case; confirm
  prevents accidental forking onto the wrong base.
- **Consequences:** Slightly more UI than a bare confirm; `promptBaseBranch`
  helper computes frequency. Testable via the typed error (CLI test can mock).
- **Status:** proposed

## 7. Pseudocode and Key Flows

### 7.1 Interactive fork of a divergent workspace

```text
$ wsm fork ttc-admin-chat deploy-dev-indexer
  (status: coinvault=task/deploy-dev-indexer, geppetto=task/deploy-dev-indexer,
           goldeneaglecoin.com=task/deploy-image, ...)
  Plan -> ErrBranchDivergence{Branches:{...}, Expected:"task/deploy-dev-indexer"}
  CLI prompt:
    "Source repos are on different branches. Choose the base branch to fork from:"
      [task/deploy-dev-indexer]  (default, most frequent)
       task/deploy-image
    "Fork using base branch 'task/deploy-dev-indexer'?"
      (shows: goldeneaglecoin.com -> task/deploy-image, rest -> task/deploy-dev-indexer)
    [y] -> req.BaseBranch = "task/deploy-dev-indexer"; Plan again; Fork
  -> Workspace forked.
```

### 7.2 Non-interactive fork of a divergent workspace

```text
$ wsm fork ttc-admin-chat deploy-dev-indexer --base-branch task/deploy-dev-indexer
  req.BaseBranch = "task/deploy-dev-indexer"
  Plan -> uses it directly (no uniformity check)
  Fork -> success

$ wsm fork ttc-admin-chat deploy-dev-indexer   (glaze mode, no flag)
  Plan -> ErrBranchDivergence
  !allowPrompt -> error: "source repos are on different branches; pass
  --base-branch (branches: task/deploy-dev-indexer, task/deploy-image)"
```

### 7.3 Flow diagram

```text
   fork.go execute(ctx, vals, emitHuman, allowPrompt)
            │
            ▼
   workflow.Plan(req)   ── uniform? ──► plan (base = the single branch)
            │ divergent & no BaseBranch
            ▼
   *ErrBranchDivergence {Branches, Expected}
            │
            ├── allowPrompt? ── no ──► error: "pass --base-branch (branches: …)"
            │
            ▼  yes
   promptBaseBranch: Select(observed + expected, default=most-frequent) + Confirm
            │
            ├── cancel ──► forkExecutionResult{Cancelled: true}
            ▼  confirm
   req.BaseBranch = chosen; workflow.Plan(req) ──► plan; workflow.Fork(req)
```

## 8. Implementation Phases

### Phase F1 — Workflow: typed error + BaseBranch override

- `pkg/wsm/workflows/fork_workflow.go`:
  - Add `BaseBranch string` to `ForkRequest`.
  - Add `ErrBranchDivergence` type with `Branches`, `Expected`, `Source`,
    `Error()`, `DistinctBranches()`.
  - In `Plan`: if `req.BaseBranch != ""`, use it; else detect; on divergence
    return `&ErrBranchDivergence{...}`.
- Test (`fork_workflow_test.go`): uniform → plan uses that branch; divergent →
  returns `*ErrBranchDivergence` with correct map; `BaseBranch` set → uses it and
  skips the check.

### Phase F2 — CLI: flag, gating, prompt

- `cmd/wsm/cmds/workspace/fork.go`:
  - Add `BaseBranch` to `ForkSettings` + the `--base-branch` flag.
  - Thread `emitHuman`/`allowPrompt` through `execute` (matching delete):
    `Run()`→`(true,true)`, `RunIntoGlazeProcessor()`→`(false,false)`.
  - On `errors.As(err, &div)`: if `!allowPrompt`, require `--base-branch`; else
    call `promptBaseBranch` and re-plan with the chosen base.
  - Add `promptBaseBranch` + `showDivergence` helpers (in `fork.go` or
    `helpers.go`).
- Test: a fake-workflow or table test asserting the glaze-mode error message
  lists branches; interactive path tested via the typed-error contract.

### Phase F3 — Validate

- Manual: reproduce the original error on a divergent workspace, confirm the
  prompt appears and the fork succeeds with the chosen base.
- `wsm fork ... --base-branch X` in glaze mode succeeds without a prompt.
- `go test ./...` + `golangci-lint run`.

## 9. Test Strategy

- **Unit (workflow):** `Plan` uniform → uses branch; divergent →
  `*ErrBranchDivergence` with `Branches`/`Expected`; `BaseBranch` set → uses
  it, no divergence error even when repos differ.
- **Unit (CLI, non-interactive):** divergent source + glaze mode + no flag →
  error contains `--base-branch` and lists distinct branches; with flag → no
  error.
- **Unit (helper):** `DistinctBranches()` returns sorted unique branches;
  most-frequent computation picks the majority.
- **Manual:** the real `wsm fork ttc-admin-chat deploy-dev-indexer` flow.

Run:

```bash
go test ./pkg/wsm/workflows/... ./cmd/wsm/cmds/workspace/... -count=1 -run 'Fork|Diverge'
go test ./... -count=1
golangci-lint run ./pkg/wsm/workflows/... ./cmd/wsm/cmds/workspace/...
```

## 10. Risks, Alternatives, Open Questions

### Risks

- **`huh` in non-TTY:** `huh.NewForm().Run()` fails without a TTY; the
  `allowPrompt` gate prevents calling it in glaze mode (mirrors delete). The
  human `Run()` path assumes a TTY — acceptable, same as delete/create.
- **Re-plan cost:** calling `Plan` twice on divergence re-runs
  `GetWorkspaceStatus`. Cheap (local git), and only on the divergent path.
- **Most-frequent tie:** two branches with equal counts → pick the
  conventional `Expected` if present, else the sorted-first. Documented; not
  critical (the user explicitly chooses via Select).

### Alternatives considered

- **Auto-pick majority branch silently.** Rejected (F2): hides a meaningful
  choice; not auditable in CI.
- **Per-repo base at fork time.** Out of scope: the fork model uses one base
  for all repos; per-repo divergence is the user's to resolve (checkout the
  desired branch before forking, or use the status per-repo override from
  design-doc 02 after the fork).

### Open questions

- Should the prompt offer a free-text "other…" option for a branch not currently
  checked out anywhere? Lean: no initially (the base should be a real ref;
  `ResolveBaseRef` from doc 02 will validate it at status time anyway).
- Should `wsm fork` warn (not block) when the chosen base has no remote-tracking
  ref (the doc-01 forked-workspace condition)? Lean: yes, as a follow-up tied to
  the status `BASE` column — defer to E6.

## 11. References (key files)

| File | Why it matters |
| --- | --- |
| `pkg/wsm/workflows/fork_workflow.go:52-110` | `ForkWorkflow.Plan` — the hard-fail to replace; add `BaseBranch` + `ErrBranchDivergence` |
| `pkg/wsm/workflows/fork_workflow.go:19-26` | `ForkRequest` — add `BaseBranch` |
| `pkg/wsm/workflows/create_workflow.go:64-72` | `BuildWorkspaceBranch` — the `task/<name>` convention for `Expected` |
| `cmd/wsm/cmds/workspace/fork.go:148-195` | `ForkCommand.execute` — add gating + prompt + re-plan |
| `cmd/wsm/cmds/workspace/fork.go:27-39` | `ForkSettings` — add `BaseBranch` |
| `cmd/wsm/cmds/workspace/delete.go:112,170-189` | `allowPrompt` gate + `huh.NewConfirm` pattern to mirror |
| `cmd/wsm/cmds/workspace/helpers.go:13-22` | `isUserCancelledError` — reuse for cancel handling |
| `pkg/wsm/status.go:119-160` | `getRepositoryStatusWithClient` — source of the `CurrentBranch` values `Plan` reads |

### Glossary

- **Base branch (fork):** the upstream branch the fork is created from; today
  derived from the source repos' current branch, now overridable via
  `--base-branch`.
- **Divergence:** source repos not all on the same branch.
- **Conventional branch:** `task/<workspace-name>` from `BuildWorkspaceBranch`;
  used as the `Expected` value in the divergence prompt.
- **`allowPrompt`:** true in human `Run()` mode (TTY prompt allowed), false in
  `RunIntoGlazeProcessor()` (glaze/JSON, flag required instead).
