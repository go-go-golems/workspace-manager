# Changelog

## 2026-03-01

- Initial workspace created

## 2026-03-01

Planning phase: added CLI-only backend migration design doc, granular task list, and initial detailed diary entry for task-driven execution.

## 2026-03-01

Implementation phase: removed gogit/hybrid backends and backend selection, standardized runtime on CLI git backend, updated tests/helpers/docs, and removed go-git module dependencies.

- Code commit: `bdbf2e6` (`wsm(git): remove gogit/hybrid and standardize on CLI backend`)
- Validation:
  - `go test ./... -count=1` (pass)
  - `golangci-lint run -v` (pass)
