var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function hasWorkspace(rows, workspaceName) {
  var i;
  for (i = 0; i < rows.length; i += 1) {
    if (rows[i].name === workspaceName) {
      return true;
    }
  }
  return false;
}

var manager = wsm.createManager({ defaultJobs: 4 });
var created = manager.workspaces.create({
  name: "ws-js-delete",
  repos: ["repo1"],
  branch: "feature/js-delete"
});
assert(created.Workspace && created.Workspace.name === "ws-js-delete", "delete test workspace was not created");

var deleted = manager.workspaces.delete({
  workspaceName: "ws-js-delete",
  removeFiles: true,
  forceWorktrees: true
});
assert(deleted.status === "deleted", "delete should return status=deleted");

var workspaces = manager.registry.listWorkspaces();
assert(!hasWorkspace(workspaces, "ws-js-delete"), "ws-js-delete should not appear after delete");

({
  ok: true,
  script: "10-workspace-delete",
  createWorkspace: created.Workspace.name,
  deleteStatus: deleted.status,
  workspacePresentAfterDelete: hasWorkspace(workspaces, "ws-js-delete")
});
