var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 3 });
var status = manager.status({ workspaceName: "ws-js-demo" });

assert(status.workspace.name === "ws-js-demo", "status should resolve ws-js-demo");
assert(status.repositories.length === 2, "expected two repositories in ws-js-demo status");

({
  ok: true,
  script: "07-default-jobs-status",
  workspace: status.workspace.name,
  repositoryCount: status.repositories.length,
  overall: status.overall
});
