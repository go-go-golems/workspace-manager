var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 2 });
var result = manager.git.rebase.run({
  workspaceName: "ws-js-rebase",
  repository: "repo1",
  targetBranch: "main",
  dryRun: true,
  jobs: 1
});

assert(result.results && result.results.length === 1, "expected one rebase result row");

({
  ok: true,
  script: "16-git-rebase-run-happy",
  rowCount: result.results.length,
  repository: result.results[0].repository,
  success: result.results[0].success,
  dryRun: result.dryRun
});
