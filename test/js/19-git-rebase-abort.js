var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 2 });
var result = manager.git.rebase.abort({
  workspaceName: "ws-js-rebase",
  repository: "repo1",
  jobs: 1
});

assert(result.rows && result.rows.length === 1, "expected one rebase abort row");

var row = result.rows[0];
var repository = row.Repository || row.repository || "";
var success = row.Success === true || row.success === true;

({
  ok: true,
  script: "19-git-rebase-abort",
  rowCount: result.rows.length,
  repository: repository,
  success: success,
  error: row.Error || row.error || ""
});
