# Tasks

## TODO

### Ticket Setup And Planning
- [x] Create ticket workspace and seed docs
- [x] Write CLI-only backend design document with implementation phases
- [x] Create granular execution task list

### Code Migration: Backend Simplification
- [ ] Replace backend selection in `BuildGitBackends` with CLI-only construction
- [ ] Remove `WSM_GIT_BACKEND` compatibility paths
- [ ] Delete `gogit` and `hybrid` git client implementations
- [ ] Delete or rewrite tests that only validate gogit/hybrid behavior

### Code Migration: Test/Helper Cleanup
- [ ] Update backend matrix tests to CLI-only assertions
- [ ] Remove integration scenario `SetBackend("hybrid")` usage
- [ ] Simplify integration helper backend API if no longer needed

### Code Migration: Docs And Dependencies
- [ ] Update architecture docs to remove hybrid/gogit references
- [ ] Remove `go-git` dependency and stale transitive entries from module files

### Validation And Delivery
- [ ] Run `go test ./... -count=1`
- [ ] Run `golangci-lint run -v`
- [ ] Update diary with each phase and exact command evidence
- [ ] Update changelog with each phase commit
- [ ] Commit phase-by-phase with focused commit messages
