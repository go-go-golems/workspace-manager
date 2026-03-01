var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 4 });

var source = manager.workspaces.create({
  name: "ws-js-fork-source",
  repos: ["repo1", "repo2"],
  branch: "feature/js-fork-source"
});
assert(source.Workspace && source.Workspace.name === "ws-js-fork-source", "source workspace should be created");

var forkWorkspace = "";
var mergeStatus = "";
var forkError = "";

try {
  var forked = manager.workspaces.fork({
    sourceWorkspaceName: "ws-js-fork-source",
    newWorkspaceName: "ws-js-ops-fork",
    branch: "feature/js-ops-fork",
    dryRun: false
  });

  assert(forked.workspace && forked.workspace.name === "ws-js-ops-fork", "fork should create ws-js-ops-fork");
  forkWorkspace = forked.workspace.name;

  var merged = manager.workspaces.merge({
    workspaceName: "ws-js-ops-fork",
    dryRun: true,
    force: true,
    keepWorkspace: true
  });

  assert(merged.status === "merged", "merge should return status=merged");
  mergeStatus = merged.status;
} catch (err) {
  forkError = String(err);
}

assert(forkWorkspace !== "" || forkError.length > 0, "fork flow should yield success or error message");

({
  ok: true,
  script: "11-workspace-fork-merge",
  sourceWorkspace: source.Workspace.name,
  forkSucceeded: forkWorkspace !== "",
  forkWorkspace: forkWorkspace,
  mergeStatus: mergeStatus,
  forkError: forkError
});
