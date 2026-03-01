var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 4 });
var viaFlat = manager.diff({ workspaceName: "ws-js-git", jobs: 2 });
var viaNamespace = manager.git.diff({ workspaceName: "ws-js-git", jobs: 2 });

assert(viaFlat.hasChanges === viaNamespace.hasChanges, "diff parity mismatch for hasChanges");

({
  ok: true,
  script: "13-git-diff",
  hasChangesFlat: viaFlat.hasChanges,
  hasChangesNamespace: viaNamespace.hasChanges,
  diffLengthFlat: (viaFlat.diff || "").length,
  diffLengthNamespace: (viaNamespace.diff || "").length,
  parity: viaFlat.hasChanges === viaNamespace.hasChanges
});
