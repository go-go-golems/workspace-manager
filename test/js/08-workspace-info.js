var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 4 });
var info = manager.workspaces.info({ workspaceName: "ws-js-ops" });
var infoField = manager.workspaces.info({ workspaceName: "ws-js-ops", field: "path" });
var infoFlat = manager.info({ workspaceName: "ws-js-ops", field: "path" });

assert(info.workspace && info.workspace.name === "ws-js-ops", "info should resolve ws-js-ops");
assert(infoField.hasField === true, "field lookup should set hasField=true");
assert(typeof infoField.value === "string" && infoField.value.length > 0, "field value should be non-empty");
assert(infoField.value === infoFlat.value, "flat and namespaced info field values should match");

({
  ok: true,
  script: "08-workspace-info",
  workspace: info.workspace.name,
  field: infoField.field,
  value: infoField.value,
  parity: infoField.value === infoFlat.value
});
