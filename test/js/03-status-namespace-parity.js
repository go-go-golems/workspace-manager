var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function repositoryNames(status) {
  var names = [];
  var rows = status.repositories || [];
  var i;
  for (i = 0; i < rows.length; i += 1) {
    names.push(rows[i].repository.name);
  }
  names.sort();
  return names;
}

function sameList(a, b) {
  if (a.length !== b.length) {
    return false;
  }
  var i;
  for (i = 0; i < a.length; i += 1) {
    if (a[i] !== b[i]) {
      return false;
    }
  }
  return true;
}

var manager = wsm.createManager({ defaultJobs: 4 });
var viaRoot = manager.status({ workspaceName: "ws-js-demo", jobs: 4 });
var viaWorkspaces = manager.workspaces.status({ workspaceName: "ws-js-demo", jobs: 4 });
var viaGit = manager.git.status({ workspaceName: "ws-js-demo", jobs: 4 });

assert(viaRoot.workspace.name === "ws-js-demo", "manager.status returned unexpected workspace");
assert(viaRoot.repositories.length === 2, "expected two repositories in ws-js-demo status");
assert(viaWorkspaces.workspace.name === "ws-js-demo", "manager.workspaces.status workspace mismatch");
assert(viaGit.workspace.name === "ws-js-demo", "manager.git.status workspace mismatch");

var rootNames = repositoryNames(viaRoot);
var workspacesNames = repositoryNames(viaWorkspaces);
var gitNames = repositoryNames(viaGit);

assert(sameList(rootNames, workspacesNames), "root/workspaces repository lists differ");
assert(sameList(rootNames, gitNames), "root/git repository lists differ");

({
  ok: true,
  script: "03-status-namespace-parity",
  workspace: viaRoot.workspace.name,
  repositoryCount: viaRoot.repositories.length,
  repositoryNames: rootNames,
  parity: {
    rootVsWorkspaces: sameList(rootNames, workspacesNames),
    rootVsGit: sameList(rootNames, gitNames)
  }
});
