var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 4 });
var ws = manager.loadWorkspace("ws-js-handle");

var name = ws.name();
var path = ws.path();
var info = ws.info();
var status = ws.status({ jobs: 2 });

assert(name === "ws-js-handle", "workspace handle should resolve ws-js-handle");
assert(typeof path === "string" && path.length > 0, "workspace handle path should be non-empty");
assert(info.workspace && info.workspace.name === "ws-js-handle", "handle info should match workspace");
assert(status.workspace && status.workspace.name === "ws-js-handle", "handle status should match workspace");

({
  ok: true,
  script: "20-workspace-handle-basics",
  name: name,
  pathLength: path.length,
  repositoryCount: status.repositories.length
});
