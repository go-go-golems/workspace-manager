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
var result = manager.git.commit({
  workspaceName: "ws-js-git",
  template: "feat",
  addAll: true,
  push: false,
  dryRun: true
});

assert(result.status === "executed" || result.status === "no_changes", "unexpected commit status");

({
  ok: true,
  script: "12-git-commit",
  status: result.status,
  message: result.message,
  selectedRepoCount: keyCount(result.selectedChanges)
});
