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
var ws = manager.loadWorkspace("ws-js-handle");

var diff = ws.git.diff({ jobs: 2 });
var log = ws.git.log({ oneline: true, limit: 10 });
var branches = ws.git.branch.list({ jobs: 2 });
var rebaseStatus = ws.git.rebase.status({ repository: "repo1", jobs: 1 });

assert(typeof diff.hasChanges === "boolean", "handle git diff should return hasChanges");
assert(keyCount(log.logs) >= 1, "handle git log should include at least one repository");
assert(branches.entries && branches.entries.length >= 1, "handle git branch list should return entries");
assert(rebaseStatus.rows && rebaseStatus.rows.length >= 1, "handle git rebase status should return rows");

({
  ok: true,
  script: "21-workspace-handle-git",
  hasChanges: diff.hasChanges,
  logRepoCount: keyCount(log.logs),
  branchEntryCount: branches.entries.length,
  rebaseRowCount: rebaseStatus.rows.length
});
