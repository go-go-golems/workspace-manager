# Tasks

## TODO


- [x] Make merged/rebase checks configurable by base branch (default main) and avoid unconditional fetch in status path
- [x] Harden CLI parsing: switch status/worktree parsing to robust machine format (porcelain -z / worktree porcelain)
- [x] Clarify/fix GitClient commit contract (return commit hash or change interface), and wire author options if retained
- [x] Expand tests: semantic status assertions for is_merged/needs_rebase, gitclient status parser coverage, worktree path-with-space coverage
- [x] Clean residual migration references (Makefile/ci docs still mentioning hybrid,gogit backends)
