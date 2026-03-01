# JS Demo Scripts for Integration Testing

This folder contains executable JavaScript examples for `wsm runner`.

Each script ends with an object literal expression so the runner can return
structured data in `--output-mode data` tests.

## Scripts

- `00-module-surface.js`: validates `require("wsm")` exports and constants.
- `01-discover-and-list.js`: runs discovery and repository listing.
- `02-create-workspace.js`: creates `ws-js-demo` with `repo1` + `repo2`.
- `03-status-namespace-parity.js`: compares `status` outputs across
  `manager`, `manager.workspaces`, and `manager.git` namespaces.
- `04-convenience-lifecycle.js`: exercises top-level convenience API calls
  (`wsm.discover`, `wsm.createWorkspace`, `wsm.status`).
- `05-list-repository-parity.js`: compares flat and grouped repository list APIs.
- `06-validation-errors.js`: demonstrates and verifies JS-side validation errors.
- `07-default-jobs-status.js`: demonstrates manager-level `defaultJobs` usage.

## Running manually

From repository root:

```bash
wsm runner test/js/00-module-surface.js
wsm runner test/js/01-discover-and-list.js --output-mode data --output json
```

The integration tests in `test/integration/scenarios/js_runner_api_scenarios_test.go`
create sandbox repositories/workspaces and execute these scripts as part of CI.
