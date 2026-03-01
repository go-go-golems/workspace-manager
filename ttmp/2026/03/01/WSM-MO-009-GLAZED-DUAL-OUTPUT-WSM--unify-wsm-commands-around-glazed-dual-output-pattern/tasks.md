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
- [x] Rewrite `wsm create` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm fork` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm merge` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm add` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm remove` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm delete` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm info` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm status` to `Run` + `RunIntoGlazeProcessor`

### Git Command Rewrites
- [x] Rewrite `wsm commit` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm diff` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm log` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm branch create` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm branch switch` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm branch list` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm rebase` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm rebase status` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm rebase continue` to `Run` + `RunIntoGlazeProcessor`
- [x] Rewrite `wsm rebase abort` to `Run` + `RunIntoGlazeProcessor`

### JS Command Rewrites
- [x] Rewrite `wsm runner` to `Run` + `RunIntoGlazeProcessor`

### Grouped Command Directory Normalization
- [x] Move `git branch` commands into `cmd/wsm/cmds/git/branch/*` with local `root.go`
- [x] Move `git rebase` commands into `cmd/wsm/cmds/git/rebase/*` with local `root.go`
- [x] Move `registry list` commands into `cmd/wsm/cmds/registry/list/*` with local `root.go`

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
