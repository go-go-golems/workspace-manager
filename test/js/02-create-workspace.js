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
  name: "ws-js-demo",
  repos: ["repo1", "repo2"],
  branch: "feature/js-demo"
});

assert(created.Workspace && created.Workspace.name === "ws-js-demo", "workspace name mismatch after create");
assert(created.FinalBranch === "feature/js-demo", "expected final branch to match explicit branch");
assert(created.Workspace.repositories && created.Workspace.repositories.length === 2, "expected two repositories in created workspace");

var workspaces = manager.workspaces.list();
assert(hasWorkspace(workspaces, "ws-js-demo"), "created workspace should appear in listWorkspaces");

({
  ok: true,
  script: "02-create-workspace",
  workspace: created.Workspace.name,
  finalBranch: created.FinalBranch,
  autoBranchGenerated: created.AutoBranchGenerated,
  listedWorkspaceCount: workspaces.length
});
