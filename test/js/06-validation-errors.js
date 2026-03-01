var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function captureError(fn) {
  try {
    fn();
    return "";
  } catch (err) {
    return String(err);
  }
}

var manager = wsm.createManager({ defaultJobs: 4 });

var missingName = captureError(function () {
  manager.createWorkspace({ repos: ["repo1"] });
});

var missingRepos = captureError(function () {
  manager.createWorkspace({ name: "ws-missing-repos" });
});

assert(missingName.indexOf("createWorkspace requires name") >= 0, "expected missing name validation message");
assert(missingRepos.indexOf("createWorkspace requires repos") >= 0, "expected missing repos validation message");

({
  ok: true,
  script: "06-validation-errors",
  missingNameMessage: missingName,
  missingReposMessage: missingRepos
});
