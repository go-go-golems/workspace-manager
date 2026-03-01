# Tasks

## TODO

### Ticket Setup And Planning
- [x] Create ticket workspace and seed docs
- [x] Write CLI-only backend design document with implementation phases
- [x] Create granular execution task list

### Code Migration: Backend Simplification
- [x] Replace backend selection in `BuildGitBackends` with CLI-only construction
- [x] Remove `WSM_GIT_BACKEND` compatibility paths
- [x] Delete `gogit` and `hybrid` git client implementations
- [x] Delete or rewrite tests that only validate gogit/hybrid behavior

### Code Migration: Test/Helper Cleanup
- [x] Update backend matrix tests to CLI-only assertions
- [x] Remove integration scenario `SetBackend("hybrid")` usage
- [x] Simplify integration helper backend API if no longer needed

### Code Migration: Docs And Dependencies
- [x] Update architecture docs to remove hybrid/gogit references
- [x] Remove `go-git` dependency and stale transitive entries from module files

### Validation And Delivery
- [x] Run `go test ./... -count=1`
- [x] Run `golangci-lint run -v`
- [x] Update diary with each phase and exact command evidence
- [x] Update changelog with each phase commit
- [x] Commit phase-by-phase with focused commit messages
