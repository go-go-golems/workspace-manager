var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 4 });
var createResult = manager.git.branch.create({
  workspaceName: "ws-js-git",
  branchName: "feature/js-branch",
  track: false
});

var switchResult = manager.git.branch.switch({
  workspaceName: "ws-js-git",
  branchName: "feature/js-branch"
});

var listResult = manager.git.branch.list({
  workspaceName: "ws-js-git",
  jobs: 2
});

assert(createResult.results && createResult.results.length >= 1, "branch create should return repository results");
assert(switchResult.results && switchResult.results.length >= 1, "branch switch should return repository results");
assert(listResult.entries && listResult.entries.length >= 1, "branch list should return entries");

({
  ok: true,
  script: "15-git-branch-create-switch-list",
  createResultCount: createResult.results.length,
  switchResultCount: switchResult.results.length,
  listEntryCount: listResult.entries.length,
  targetBranch: "feature/js-branch"
});
