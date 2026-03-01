var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function keyCount(obj) {
  return Object.keys(obj || {}).length;
}

var manager = wsm.createManager({ defaultJobs: 4 });

var infoFlat = manager.info({ workspaceName: "ws-js-handle", field: "path" });
var infoNs = manager.workspaces.info({ workspaceName: "ws-js-handle", field: "path" });

var reposFlat = manager.listRepositories({ tags: [] });
var reposNs = manager.registry.listRepositories({ tags: [] });

var wsFlat = manager.listWorkspaces();
var wsNs = manager.registry.listWorkspaces();

var diffFlat = manager.diff({ workspaceName: "ws-js-handle", jobs: 2 });
var diffNs = manager.git.diff({ workspaceName: "ws-js-handle", jobs: 2 });

var logFlat = manager.log({ workspaceName: "ws-js-handle", oneline: true, limit: 10 });
var logNs = manager.git.log({ workspaceName: "ws-js-handle", oneline: true, limit: 10 });

var parity = {
  info: infoFlat.value === infoNs.value,
  repositories: reposFlat.length === reposNs.length,
  workspaces: wsFlat.length === wsNs.length,
  diff: diffFlat.hasChanges === diffNs.hasChanges,
  log: keyCount(logFlat.logs) === keyCount(logNs.logs)
};

assert(parity.info, "flat vs namespaced info parity failed");
assert(parity.repositories, "flat vs namespaced repositories parity failed");
assert(parity.workspaces, "flat vs namespaced workspaces parity failed");
assert(parity.diff, "flat vs namespaced diff parity failed");
assert(parity.log, "flat vs namespaced log parity failed");

({
  ok: true,
  script: "22-flat-vs-namespace-parity-extended",
  parity: parity,
  repositoryCount: reposFlat.length,
  workspaceCount: wsFlat.length,
  logRepoCount: keyCount(logFlat.logs)
});
