### 1. Introduction

This report analyzes the `workspace-manager` codebase with a focus on its command line interfaces (CLI), inner architecture, used modules, and control flow. The goal is to equip developers to consolidate, refactor, and extend the system. The analysis proceeds by examining the directory structure, enumerating commands and their handlers, inspecting core business logic in `pkg/wsm`, and mapping cross-cutting flows like workspace creation, discovery, status, and git operations.

---

### 2. Codebase Overview

**Reasoning:**
- The project is a Go monorepo module: `module github.com/go-go-golems/workspace-manager` (see `workspace-manager/go.mod`). The primary binary is a Cobra CLI exposed via `cmd/wsm` and concrete subcommands defined under `cmd/cmds`.
- The business logic lives in `pkg/wsm`, which is cleanly separated from the CLI layer, enabling testing and reuse.
- Output formatting helpers are in `pkg/output`, decoupling user-facing messages from logging.
- The CLI `root` initialization wires configuration via `clay.InitViper`, logging via `glazed` logging helper, and completion with `carapace`.
- A rich set of subcommands exists (create, fork, merge, add/remove/delete, list/info/status, sync/push/rebase/branch/diff/log, pr, tmux, starship). Each file `cmd_*.go` typically exposes `func New<Name>Command() *cobra.Command` and a `RunE` that calls into `pkg/wsm`.

**Conclusion:**
- Top-level layout (selected):
  - `cmd/wsm/root.go`: Cobra root command, configuration, and subcommand registration; entrypoint exposed via `Execute()`.
  - `cmd/cmds/*.go`: Subcommand constructors and handlers (Cobra).
  - `pkg/wsm/*.go`: Core domain logic: discovery, workspace lifecycle, git operations, status, synchronization, utility methods.
  - `pkg/output/styles.go`: Styled user-facing output helpers.
  - `README.md`, `IMPLEMENTATION.md`: High-level docs and an internal implementation guide.

---

### 3. Command Line Interfaces

**Reasoning:**
- Entry point is `cmd/wsm/root.go`. The `rootCmd` sets `Use: "wsm"`, initializes logging (`logging.InitLoggerFromViper`) and configuration (`clay.InitViper("workspace-manager", rootCmd)`), and registers all subcommands. Completion is enabled with `carapace.Gen(rootCmd)`.
- Registered commands (from `root.go`): `NewDiscoverCommand`, `NewListCommand`, `NewCreateCommand`, `NewForkCommand`, `NewMergeCommand`, `NewAddCommand`, `NewRemoveCommand`, `NewDeleteCommand`, `NewInfoCommand`, `NewStatusCommand`, `NewPRCommand`, `NewPushCommand`, `NewCommitCommand`, `NewSyncCommand`, `NewBranchCommand`, `NewRebaseCommand`, `NewDiffCommand`, `NewLogCommand`, `NewTmuxCommand`, `NewStarshipCommand`.
- Each subcommand file in `cmd/cmds` builds a `*cobra.Command` with flags/args and a `RunE` that composes `pkg/wsm` services. Examples observed via grep:
  - `cmd_sync.go` defines `sync` with subcommands `all`, `pull`, `push`, delegating to `runSyncAll`, `runSyncPull`, `runSyncPush`.
  - `cmd_status.go` defines `status [workspace-name]` and resolves workspace before delegating status gathering to `wsm.StatusChecker`.
  - `cmd_push.go` defines `push <remote-name> [workspace-name]` and calls push routine across repos.
  - `cmd_rebase.go` defines `rebase [repository]` with optional filtering.
  - `cmd_list.go` defines `list` with `repos` and `workspaces` subcommands; list routines read registry/workspace configs.
  - `cmd_tmux.go` integrates tmux session management for a workspace.
  - `cmd_starship.go` generates starship prompt configuration.
- The commands rely on shared workspace discovery/loading helpers inside `pkg/wsm` (e.g., `LoadWorkspaces`, `LoadWorkspace`, or status-based detection in `IMPLEMENTATION.md`).

**Conclusion:**
- CLI Commands and mappings (selected):
  - `discover` → `cmd/cmds/cmd_discover.go` → `wsm.RepositoryDiscoverer.DiscoverRepositories`.
  - `create` → `cmd/cmds/cmd_create.go` → `wsm.WorkspaceManager.CreateWorkspace`.
  - `fork` → `cmd/cmds/cmd_fork.go` → `wsm.WorkspaceManager.CreateWorkspace` using source workspace metadata and base branch.
  - `merge` → `cmd/cmds/cmd_merge.go` → uses `wsm.StatusChecker`, repository loop merge, then `wsm.WorkspaceManager.DeleteWorkspace` optionally.
  - `add` → `cmd/cmds/cmd_add.go` → `wsm.WorkspaceManager.AddRepositoryToWorkspace`.
  - `remove` → `cmd/cmds/cmd_remove.go` → `wsm.WorkspaceManager.RemoveRepositoryFromWorkspace`.
  - `delete` → `cmd/cmds/cmd_delete.go` → `wsm.WorkspaceManager.DeleteWorkspace`.
  - `list repos|workspaces` → `cmd/cmds/cmd_list.go` → `wsm.RepositoryDiscoverer.GetRepositories*`, `wsm.LoadWorkspaces`.
  - `info` → `cmd/cmds/cmd_info.go` → `wsm.WorkspaceManager.LoadWorkspace` and derived views.
  - `status` → `cmd/cmds/cmd_status.go` → `wsm.NewStatusChecker().GetWorkspaceStatus`.
  - `commit` → `cmd/cmds/cmd_commit.go` → `wsm.NewGitOperations(workspace).CommitChanges`.
  - `push` → `cmd/cmds/cmd_push.go` → repository push across workspace.
  - `sync all|pull|push` → `cmd/cmds/cmd_sync.go` → pull/push/rebase orchestration.
  - `branch`, `rebase`, `diff`, `log` → related git orchestration using `wsm.GitOperations` and helpers.
  - `pr` → `cmd/cmds/cmd_pr.go` → pull request creation routine (implementation likely shells out to gh/forge tooling; verify in file).
  - `tmux` → `cmd/cmds/cmd_tmux.go` → session creation/attach.
  - `starship` → `cmd/cmds/cmd_starship.go` → outputs starship prompt config.

---

### 4. Inner Architecture

**Reasoning:**
- Central domain package: `pkg/wsm`.
  - `types.go` defines core data structures: `Repository`, `RepositoryRegistry`, `Workspace`, `WorkspaceConfig`, `RepositoryStatus`, `WorkspaceStatus`, and `WorktreeInfo`.
  - `workspace.go` defines `WorkspaceManager` with configuration resolution, workspace creation, go.work generation, AGENT.md copying, metadata creation (`.wsm/wsm.json`), setup script execution, rollback and cleanup, adding/removing repositories, and deletion workflows. Many steps print user-facing info via `pkg/output` and wrap errors with `github.com/pkg/errors`.
  - `discovery.go` defines `RepositoryDiscoverer` for scanning directories, analyzing git repos, extracting metadata, categorization, and registry persistence at `~/.config/workspace-manager/registry.json`.
  - `status.go` defines `StatusChecker` to compute per-repository and overall workspace status, including ahead/behind, conflicts, merge/rebase state (via `CheckBranchMerged`/`CheckBranchNeedsRebase` symbols referenced in `status.go`—these are expected in `sync_operations.go` or `git_utils.go`).
  - `git_operations.go` provides higher-level git orchestration for multi-repo operations: staging, committing, pushing, and diff generation with per-repo execution.
  - Additional files: `git_utils.go`, `sync_operations.go`, and `utils.go` include helpers for git checks, synchronization, and execution of setup tasks.
- Cross-cutting concerns:
  - User feedback via `pkg/output/styles.go` for consistent, styled messages.
  - Error handling standardized with `errors.Wrap[f]` and contextual strings.
  - Configuration defaults resolved in `loadConfig()` based on user home and config dirs.
  - Metadata and setup automation provide a predictable workspace structure and environment variables for scripts.

Pseudocode overview of key objects and interactions:
```
WorkspaceManager:
  NewWorkspaceManager():
    cfg := loadConfig()
    rd := NewRepositoryDiscoverer(cfg.RegistryPath)
    rd.LoadRegistry()
    return &WorkspaceManager{config: cfg, Discoverer: rd, workspaceDir: cfg.WorkspaceDir}

  CreateWorkspace(ctx, name, repoNames, branch, baseBranch, agentSource, dryRun):
    repos := FindRepositories(repoNames)
    ws := Workspace{Name: name, Path: join(workspaceDir, name), Repositories: repos, Branch: branch, BaseBranch: baseBranch, GoWorkspace: shouldCreateGoWorkspace(repos), AgentMD: agentSource}
    if dryRun return ws
    createWorkspaceStructure(ctx, ws)
    SaveWorkspace(ws)
    return ws

RepositoryDiscoverer:
  DiscoverRepositories(ctx, paths, recursive, maxDepth):
    for p in paths: repos += scanDirectory(ctx, p, recursive, maxDepth, 0)
    registry.Repositories = merge(existing, repos); registry.LastScan = now
    SaveRegistry()

StatusChecker:
  GetWorkspaceStatus(ctx, ws):
    for repo in ws.Repositories: statuses += getRepositoryStatus(ctx, repo, join(ws.Path, repo.Name))
    return WorkspaceStatus{Workspace: ws, Repositories: statuses, Overall: calculateOverallStatus(statuses)}

GitOperations:
  CommitChanges(ctx, op):
    for repo, files in op.Files:
      stage as needed; if hasStagedChanges -> commit; collect successes
    if op.Push: push successes
```

**Conclusion:**
- Modules and responsibilities:
  - `pkg/wsm/workspace.go` (`WorkspaceManager`): workspace lifecycle, worktrees, metadata, setup scripts, add/remove/delete operations.
  - `pkg/wsm/discovery.go` (`RepositoryDiscoverer`): scanning, analysis, registry IO.
  - `pkg/wsm/status.go` (`StatusChecker`): per-repo git state aggregation and overall health.
  - `pkg/wsm/git_operations.go` (`GitOperations`): multi-repo staging/commit/push/diff.
  - `pkg/wsm/types.go`: shared data contracts.
  - `pkg/output/styles.go`: presentation helpers for CLI output.

---

### 5. Used Modules and Dependencies

**Reasoning:**
- From `go.mod`:
  - Core CLI: `github.com/spf13/cobra` for command definitions; `github.com/carapace-sh/carapace` for completion.
  - Config/logging: `github.com/go-go-golems/clay` for Viper setup; `github.com/go-go-golems/glazed` for logging bootstrap; `github.com/rs/zerolog` for structured logs.
  - TUI/prompt: `github.com/charmbracelet/huh` for interactive selections during branch conflicts in worktree creation; `github.com/charmbracelet/lipgloss` for styled output.
  - Error handling: `github.com/pkg/errors` for wrapping.
  - Indirects include `bubbletea`, `bubbles`, `viper`, `pflag`, etc., due to upstreams.
- Within code:
  - `workspace.go` uses `huh` for interactive branch-handling decisions and `os/exec` for git commands; prints via `pkg/output` and `fmt`.
  - `discovery.go`, `status.go`, `git_operations.go` consistently shell out to `git`.

**Conclusion:**
- External dependencies and roles:
  - `cobra`: CLI structure and parsing.
  - `carapace`: dynamic shell completions.
  - `clay` + `glazed`: configuration and logging initialization.
  - `zerolog`: structured runtime logs (initialized at root; actual `pkg/output` prints are user-oriented).
  - `huh`: interactive user prompts during branch existence decisions.
  - `lipgloss`: color/styling for output.
  - `pkg/errors`: contextual error propagation.

---

### 6. Control Flow

**Reasoning:**
- Workspace creation (`wsm.WorkspaceManager.CreateWorkspace`):
  1) Validate name; 2) Resolve repositories via registry (`FindRepositories` → `Discoverer.GetRepositories()`); 3) Build `Workspace{}`; 4) If `dryRun`, return; 5) `createWorkspaceStructure`:
     - `mkdir workspace.Path`.
     - Loop `Repositories`: `createWorktree` with branch logic:
       - If `Branch == ""` → `git worktree add target`
       - Else check `CheckBranchExists` and `CheckRemoteBranchExists`:
         - If local branch exists → interactive choice via `huh` to overwrite/use/cancel.
         - Else if remote exists → `git worktree add -b Branch target origin/Branch`.
         - Else if `BaseBranch != ""` → `git worktree add -b Branch target BaseBranch`; otherwise new branch from HEAD.
       - On failure: rollback created worktrees and cleanup workspace directory.
     - If Go workspace → `CreateGoWorkspace` writes `go.work` using repo subpaths containing `go.mod`.
     - If `AgentMD` set → copy file to workspace root.
     - Create `.wsm/wsm.json` metadata.
     - Execute setup scripts: root `.wsm/setup.sh` then aggregated `.wsm/setup.d/*` and per-repo `.wsm/setup.d/*` in lexical order.
     - Non-fatal warnings are logged for failures in metadata/setup steps.
     - Finally, `SaveWorkspace` persists config JSON to `~/.config/workspace-manager/workspaces/<name>.json`.

- Repository discovery (`wsm.RepositoryDiscoverer.DiscoverRepositories`):
  - For each input path, recursively `scanDirectory` up to `maxDepth` honoring skip rules; if directory is a repo, extract metadata (`analyzeRepository` using git commands), categorize (presence of common files/dirs), and merge into registry by `Path`.
  - Save registry to the configured JSON path.

- Status check (`wsm.StatusChecker.GetWorkspaceStatus`):
  - For each repo worktree path under the workspace, gather branch, modified/staged/untracked files, ahead/behind relative to upstream, merge/rebase needs (via helpers), and compute an overall state string among `conflicts`, `modified`, `needs-sync`, `clean`.

- Multi-repo commit/push/diff (`wsm.GitOperations`):
  - `GetWorkspaceChanges` gathers `git status --porcelain` for each repo into a `map[string][]FileChange`.
  - `CommitChanges` stages all or selected files, commits if staged changes exist, and optionally pushes successes.
  - `GetDiff` builds a labeled concatenation of per-repo diffs (staged or working tree).

- Deletion and cleanup (`wsm.WorkspaceManager.DeleteWorkspace`):
  - Load workspace, remove each worktree (`git worktree remove [--force] <worktreePath>`), optionally delete the workspace directory (with logging of planned deletions), clean workspace-specific files otherwise, then remove the stored JSON config.

Pseudocode for non-trivial flows:
```
createWorktree(ctx, ws, repo):
  target := join(ws.Path, repo.Name)
  if ws.Branch == "":
    return run("git worktree add", target)
  local := CheckBranchExists(repo.Path, ws.Branch)
  remote := CheckRemoteBranchExists(repo.Path, ws.Branch)
  if local:
    choice := huh.Select(overwrite|use|cancel)
    if choice == overwrite:
      if remote: add -B Branch target origin/Branch
      else if ws.BaseBranch != "": add -B Branch target BaseBranch
      else: add -B Branch target
    if choice == use: add target Branch
    if choice == cancel: error
  else:
    if remote: add -b Branch target origin/Branch
    else if ws.BaseBranch != "": add -b Branch target BaseBranch
    else: add -b Branch target
```

**Conclusion:**
- Key flows integrate CLI→manager→git shell commands with robust user feedback, interactive decision points, and rollback/cleanup safeguards.
- Status and commit flows isolate git plumbing in `wsm` and avoid duplicating logic in CLI layer.

---

### 7. Notable Issues, Weaknesses, or Redundancies

**Reasoning:**
- Interactive prompts: `createWorktree` and removal flows sometimes block on user input (`huh` form, `fmt.Scanln`). For automation and non-interactive CI usage, provide `--yes`/`--force`/`--no-interactive` flags to avoid blocking.
- Concurrency: Repository operations are sequential in several loops (create worktrees, commit/push, status). Introducing controlled concurrency (with bounded workers and ordered output) could speed up operations while preserving logging order. Consider `errgroup`.
- Error surface: `ExecuteWorktreeCommand` takes `...args` including the binary name; consider a dedicated helper accepting `name string, args ...string` to avoid accidental malformed invocations. Also consider centralizing git command execution and retry/backoff for transient errors.
- Configuration hard-coding: `CreateGoWorkspace` writes `go 1.23`; ensure this aligns with actual minimum supported Go and possibly detect from `go.mod` files.
- Duplicate branch existence checks exist in both `workspace.go` and `CreateWorktreeForAdd`; unify via shared helper to reduce drift.
- Status advanced checks: `CheckBranchMerged` and `CheckBranchNeedsRebase` are referenced in `status.go` but their definitions are not visible in the inspected files; ensure they exist in `git_utils.go` or `sync_operations.go` and are tested.
- Output/log separation: `pkg/output.LogInfo` currently ignores structured fields; consider wiring these to `zerolog` for richer logs while keeping pretty prints for users.
- Registry merge key: `mergeRepositories` uses `repo.Path` as key, while lookups for `FindRepositories` key by `repo.Name` later; ensure global uniqueness or add safeguards for duplicate names.
- Deletion safety: Worktree removal prompts for confirmation when forced untracked deletions are needed; provide a non-interactive override for scripts.

**Conclusion:**
- Recommended refactors:
  - Add global `--non-interactive` and per-command `--yes/--force` to bypass prompts.
  - Introduce a `GitRunner` with context propagation, logging, retries, and consistent arg handling; reuse across discover/status/git ops.
  - Use `errgroup` to parallelize per-repo operations with a concurrency limit.
  - Deduplicate branch-existence logic; centralize in `git_utils.go`.
  - Detect `go` version and modules dynamically for `go.work`.
  - Ensure `CheckBranchMerged`/`CheckBranchNeedsRebase` exist and are consistent.
  - Enhance `pkg/output` to also log structured fields through `zerolog`.

---

### 8. CLI Completers and Developer UX

**Reasoning:**
- `carapace.Gen(rootCmd)` enables auto-generated completion; dedicated completion helpers are likely in `cmd/cmds/completion_helpers.go` to provide dynamic workspace/repo/tag completion (confirmed by README description).
- Commands such as `list repos --tags` depend on discovering categories/tags from `Repository.Categories`; completion can source tags by scanning registry.

**Conclusion:**
- Ensure completion helper functions in `cmd/cmds/completion_helpers.go` surface:
  - Workspace names from `~/.config/workspace-manager/workspaces/*.json`.
  - Repository names from registry.
  - Tags from union of `Repository.Categories`.

---

### 9. Data and Persistence Model

**Reasoning:**
- Registry: `~/.config/workspace-manager/registry.json` with `RepositoryRegistry{Repositories, LastScan}`.
- Workspaces: `~/.config/workspace-manager/workspaces/<name>.json` persisting `Workspace` state.
- On-disk workspaces: user home directory under `~/workspaces/YYYY-MM-DD/<name>/` with per-repo worktrees and `.wsm` metadata.
- Metadata file: `.wsm/wsm.json` includes workspace and repo details plus exported environment variables.

**Conclusion:**
- Persistence relies on simple JSON files in per-user config directories; this is straightforward and easy to inspect and backup. Consider schema versioning if evolving fields.

---

### 10. Edge Cases and Unclear/Unused Modules

**Reasoning:**
- Unclear references: `CheckBranchMerged` and `CheckBranchNeedsRebase` used in `status.go` are not in the files inspected here; they likely live in `pkg/wsm/git_utils.go` or `pkg/wsm/sync_operations.go`. If missing, status summaries will degrade.
- `pkg/wsm/utils.go` and `pkg/wsm/sync_operations.go` were not opened during this pass; they likely implement synchronization logic (pull, push, rebase) referenced by CLI `sync` subcommands.
- README lists `log`, `branch`, `diff`, `commit`, `pr` commands; ensure the corresponding command files exist and are wired in `root.go`. We saw `cmd_diff.go`, `cmd_rebase.go`, `cmd_pr.go`, etc., via grep.

**Conclusion:**
- Action: verify `git_utils.go` and `sync_operations.go` for the missing helpers and document their usage; ensure tests cover the status in presence/absence of upstream.

---

### 11. Suggested Next Steps for Refactoring

**Reasoning:**
- Based on the architecture and issues noted, a consolidation pass can improve maintainability and robustness.

**Conclusion:**
- High-priority tasks:
  - Unify git command execution under a single `GitRunner` with consistent arg building and logging, and a `--non-interactive` mode.
  - Add concurrency for per-repo ops via `errgroup` with a `--jobs` flag.
  - Normalize branch existence and creation logic in one place and reuse from `createWorktree` and `CreateWorktreeForAdd`.
  - Make `go.work` creation detect version and modules dynamically.
  - Expand completion helpers to cover all dynamic entities and ensure they fail gracefully when registry/workspaces are empty.
  - Introduce structured logging fields wiring in `pkg/output` to `zerolog` while preserving styled prints.
  - Add schema versioning to persisted JSON files.

---

### 12. Appendix: File and Symbol References

- CLI root: `cmd/wsm/root.go` (`rootCmd`, `Execute`, `init` registration).
- Commands (constructors): `cmd/cmds/*.go` (e.g., `NewSyncCommand`, `NewStatusCommand`, `NewPushCommand`, `NewRebaseCommand`, `NewListCommand`, `NewTmuxCommand`, `NewStarshipCommand`).
- Core domain:
  - `pkg/wsm/types.go`: `Repository`, `RepositoryRegistry`, `Workspace`, `WorkspaceConfig`, `RepositoryStatus`, `WorkspaceStatus`, `WorktreeInfo`.
  - `pkg/wsm/workspace.go`: `WorkspaceManager`, `CreateWorkspace`, `CreateGoWorkspace`, `AddRepositoryToWorkspace`, `RemoveRepositoryFromWorkspace`, `DeleteWorkspace`, `LoadWorkspace`, `LoadWorkspaces`, `ExecuteWorktreeCommand`, `CheckBranchExists`, `CheckRemoteBranchExists`, setup and metadata helpers.
  - `pkg/wsm/discovery.go`: `RepositoryDiscoverer`, `DiscoverRepositories`, `scanDirectory`, `analyzeRepository`, `categorizeRepository`, `SaveRegistry`, `LoadRegistry`.
  - `pkg/wsm/status.go`: `StatusChecker`, `GetWorkspaceStatus`, `getRepositoryStatus`, `calculateOverallStatus`.
  - `pkg/wsm/git_operations.go`: `GitOperations`, `GetWorkspaceChanges`, `CommitChanges`, `GetDiff` and helpers.
- Output: `pkg/output/styles.go`: `PrintError`, `PrintSuccess`, `PrintInfo`, `PrintWarning`, `PrintHeader`, `Spinner`, `LogInfo/Warn/Error`.
