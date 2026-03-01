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
var viaFlat = manager.log({
  workspaceName: "ws-js-git",
  oneline: true,
  limit: 10
});
var viaNamespace = manager.git.log({
  workspaceName: "ws-js-git",
  oneline: true,
  limit: 10
});

assert(keyCount(viaFlat.logs) >= 1, "expected at least one repository log");
assert(keyCount(viaFlat.logs) === keyCount(viaNamespace.logs), "flat/namespace log repo count mismatch");

({
  ok: true,
  script: "14-git-log",
  repoCountFlat: keyCount(viaFlat.logs),
  repoCountNamespace: keyCount(viaNamespace.logs),
  parity: keyCount(viaFlat.logs) === keyCount(viaNamespace.logs)
});
