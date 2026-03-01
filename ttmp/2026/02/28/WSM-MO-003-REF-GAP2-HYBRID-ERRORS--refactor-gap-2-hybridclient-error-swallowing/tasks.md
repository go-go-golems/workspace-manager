# Tasks

## Done

- [x] Create ticket workspace and baseline docs
- [x] Collect line-anchored evidence from hybrid client and backend wiring
- [x] Create and run reproducible experiment script
- [x] Write detailed bug report and fix design for intern handoff
- [x] Refactor `HybridClient` mutating methods to propagate real errors
- [x] Replace `== ErrNotImplemented` checks with `errors.Is`
- [x] Add shared fallback helper(s) to remove copy-paste logic
- [x] Add table-driven unit tests for all affected mutating methods
- [x] Add regression tests explicitly asserting primary real-error propagation
- [x] Re-run repro script and capture fixed output
- [x] Run `go test ./pkg/wsm/gitclient` and broader validation target

## Validation Notes

- Targeted tests passed:
  - `go test ./pkg/wsm/gitclient -run 'Hybrid|RemoteBranch' -v`
- Broader run executed:
  - `go test ./...` still fails in integration scenarios due existing harness/stale-binary issues outside this ticket scope.
