var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 2 });
var result = manager.git.rebase.status({
  workspaceName: "ws-js-rebase",
  repository: "repo1",
  jobs: 1
});

assert(result.rows && result.rows.length === 1, "expected one rebase status row");

var row = result.rows[0];
var repository = row.Repository || row.repository || "";

({
  ok: true,
  script: "17-git-rebase-status",
  rowCount: result.rows.length,
  repository: repository,
  state: row.State || row.state || "",
  conflicts: row.Conflicts || row.conflicts || 0
});
