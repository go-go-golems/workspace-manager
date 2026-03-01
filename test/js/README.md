# JS Demo Scripts for Integration Testing

This folder contains executable JavaScript examples for `wsm runner`.

Each script ends with an object literal expression so integration tests can
parse a structured `result` payload in `--output-mode data`.

## Script Matrix (00-22)

Baseline scripts:

- `00-module-surface.js`: validates baseline module exports and constants.
- `01-discover-and-list.js`: runs discovery and repository listing.
- `02-create-workspace.js`: creates `ws-js-demo`.
- `03-status-namespace-parity.js`: checks status parity across flat and namespaced methods.
- `04-convenience-lifecycle.js`: exercises top-level convenience wrappers.
- `05-list-repository-parity.js`: checks list parity (`flat` vs `registry`).
- `06-validation-errors.js`: validates required-field TypeError behavior.
- `07-default-jobs-status.js`: demonstrates manager `defaultJobs`.

Workspace lifecycle expansion:

- `08-workspace-info.js`: `workspaces.info` and `info` parity checks.
- `09-workspace-add-remove.js`: add/remove repository lifecycle flow.
- `10-workspace-delete.js`: create/delete workspace lifecycle flow.
- `11-workspace-fork-merge.js`: fork + dry-run merge flow.

Git core expansion:

- `12-git-commit.js`: dry-run commit path (message/template + selection summary).
- `13-git-diff.js`: diff parity (`manager.diff` vs `manager.git.diff`).
- `14-git-log.js`: log parity (`manager.log` vs `manager.git.log`).
- `15-git-branch-create-switch-list.js`: branch create/switch/list lifecycle.

Rebase expansion:

- `16-git-rebase-run-happy.js`: dry-run rebase run path.
- `17-git-rebase-status.js`: rebase status rows.
- `18-git-rebase-continue.js`: continue rows.
- `19-git-rebase-abort.js`: abort rows.

Workspace handle and parity expansion:

- `20-workspace-handle-basics.js`: `loadWorkspace` + handle basics (`name/path/info/status`).
- `21-workspace-handle-git.js`: handle-scoped git/branch/rebase methods.
- `22-flat-vs-namespace-parity-extended.js`: extended parity checks across info/list/diff/log.

## Scenario Mapping

Scripts are executed by integration scenario files:

- `test/integration/scenarios/js_runner_api_scenarios_test.go`: scripts `00-07`.
- `test/integration/scenarios/js_runner_workspace_lifecycle_scenarios_test.go`: scripts `08-11`.
- `test/integration/scenarios/js_runner_git_ops_scenarios_test.go`: scripts `12-15`.
- `test/integration/scenarios/js_runner_rebase_scenarios_test.go`: scripts `16-19`.
- `test/integration/scenarios/js_runner_workspace_handle_scenarios_test.go`: scripts `20-22`.

Each scenario sets up its own sandbox repositories/workspaces before running
scripts, then asserts:

- runner row contract (`status`, `has_result`, `result`),
- script contract (`ok: true`, stable script id),
- domain-specific expected outcomes.

## Manual Execution

From repository root:

```bash
wsm runner test/js/00-module-surface.js
wsm runner test/js/08-workspace-info.js --output-mode data --output json --print-result=false
```

Most scripts expect scenario-managed setup (specific workspaces/repos) and are
therefore intended to be run through integration scenarios rather than directly.
