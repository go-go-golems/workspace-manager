# Tasks

## TODO

### Planning And Ticket Setup
- [x] Create ticket workspace and seed docs
- [x] Audit all WSM commands against Glazed dual-output pattern
- [x] Write command-by-command implementation design document
- [x] Upload design bundle to reMarkable
- [x] Add explicit requirement: registry list human output uses concise template (no tabwriter)

### Core Migration Scaffolding
- [x] Add transitional dual-mode cobra builder helper for migrated commands (`--with-glaze-output`)
- [x] Define reusable execution helper pattern (`execute(...)`) and apply to first migrated command

### Registry Command Rewrites
- [x] Rewrite `wsm discover` to `Run` + `RunIntoGlazeProcessor` (remove `EmitRows`/`output-mode` branching)
- [x] Rewrite `wsm list repos` to `Run` + `RunIntoGlazeProcessor` (keep concise human template)
- [x] Rewrite `wsm list workspaces` to `Run` + `RunIntoGlazeProcessor` (keep concise human template)
- [x] Add/adjust registry command tests for human mode and glaze mode

### Workspace Command Rewrites
- [ ] Rewrite `wsm create` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm fork` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm merge` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm add` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm remove` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm delete` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm info` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm status` to `Run` + `RunIntoGlazeProcessor`

### Git Command Rewrites
- [ ] Rewrite `wsm commit` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm diff` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm log` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm branch create` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm branch switch` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm branch list` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm rebase` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm rebase status` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm rebase continue` to `Run` + `RunIntoGlazeProcessor`
- [ ] Rewrite `wsm rebase abort` to `Run` + `RunIntoGlazeProcessor`

### JS Command Rewrites
- [ ] Rewrite `wsm runner` to `Run` + `RunIntoGlazeProcessor`

### Grouped Command Directory Normalization
- [ ] Move `git branch` commands into `cmd/wsm/cmds/git/branch/*` with local `root.go`
- [ ] Move `git rebase` commands into `cmd/wsm/cmds/git/rebase/*` with local `root.go`
- [ ] Move `registry list` commands into `cmd/wsm/cmds/registry/list/*` with local `root.go`

### Remove Legacy Output Plumbing
- [ ] Remove `EmitRows` from `cmd/wsm/cmds/common/runtime.go`
- [ ] Remove runtime `output-mode` field/resolver helpers from migrated commands and common runtime
- [ ] Ensure command help and docs describe human default + `--with-glaze-output` usage

### Validation And Delivery
- [x] Run focused tests for rewritten command groups
- [x] Run broader command package checks (`go test ./cmd/wsm/...`)
- [x] Update diary with each migration phase (including exact failures)
- [x] Update changelog and doc relations for each phase
- [ ] Commit each migration phase with focused commits
