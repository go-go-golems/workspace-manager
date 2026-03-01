var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 4 });
var addResult = manager.workspaces.add({
  workspaceName: "ws-js-ops",
  repoName: "repo2",
  branch: "feature/js-ops-repo2",
  force: true
});

assert(addResult.status === "added", "addRepository should return status=added");

var statusAfterAdd = manager.status({ workspaceName: "ws-js-ops", jobs: 2 });
assert(statusAfterAdd.repositories.length === 2, "workspace should have 2 repositories after add");

var removeResult = manager.workspaces.remove({
  workspaceName: "ws-js-ops",
  repoName: "repo2",
  force: true,
  removeFiles: true
});

assert(removeResult.status === "removed", "removeRepository should return status=removed");

var statusAfterRemove = manager.workspaces.status({ workspaceName: "ws-js-ops", jobs: 2 });
assert(statusAfterRemove.repositories.length === 1, "workspace should have 1 repository after remove");

({
  ok: true,
  script: "09-workspace-add-remove",
  addStatus: addResult.status,
  removeStatus: removeResult.status,
  repositoryCountAfterAdd: statusAfterAdd.repositories.length,
  repositoryCountAfterRemove: statusAfterRemove.repositories.length
});
