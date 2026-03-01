# Changelog

## 2026-03-01

- Initial workspace created


## 2026-03-01

Initialized ticket for post-migration CLI git client cleanup: configurable base branch semantics, parser hardening, commit contract cleanup, and missing semantic test coverage.

### Related Files

- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/git_utils.go — Status semantics follow-up
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/cli_client.go — Git client behavior follow-up
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/test/integration/scenarios/status_diff_test.go — Coverage gap follow-up


## 2026-03-01

Task 1 complete: added configurable base branch resolution (main default), removed unconditional fetch in merged/rebase status checks, and aligned CLI/JS rebase default target behavior.

### Related Files

- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/branch/types.go — Default/configured base branch resolver
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/git_utils.go — Base-branch-aware merged/rebase checks
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/status.go — Pass workspace base branch into status semantics


## 2026-03-01

Task 2 complete: hardened git client parsing using machine formats (status --porcelain -z and worktree list --porcelain) to avoid brittle whitespace parsing.

### Related Files

- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/cli_client.go — Status parser uses NUL-delimited porcelain
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/worktree_cli.go — Worktree parser uses porcelain stanzas

## 2026-03-01

Task 3 complete: simplified GitClient commit contract by removing unused CommitOptions and changing commit signature to return only error.

### Related Files

- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/git_operations.go — Caller updated for new commit contract
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/cli_client.go — Commit implementation simplified
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/client.go — Commit interface simplified


## 2026-03-01

Task 4 complete: added semantic status regression coverage (is_merged/needs_rebase) and parser-focused gitclient tests for status/worktree edge cases including path spaces.

### Related Files

- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/pkg/wsm/gitclient/status_worktree_cli_test.go — Parser edge-case coverage
- /home/manuel/workspaces/2025-08-23/refactor-workspace-manager/workspace-manager/test/integration/scenarios/status_semantics_test.go — Semantic status assertions

