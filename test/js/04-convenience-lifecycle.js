var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var discovered = wsm.discover({ paths: ["."], recursive: true, maxDepth: 3 });
assert(discovered.RepositoryCount >= 1, "discover should find at least one repository");

var created = wsm.createWorkspace({
  name: "ws-js-convenience",
  repos: ["repo1"],
  branchPrefix: "feat"
});

assert(created.Workspace && created.Workspace.name === "ws-js-convenience", "convenience createWorkspace returned wrong workspace");
assert(created.AutoBranchGenerated === true, "expected auto-generated branch when branch omitted");
assert(created.FinalBranch === "feat/ws-js-convenience", "unexpected generated branch from branchPrefix");

var status = wsm.status({ workspaceName: "ws-js-convenience", jobs: 2 });
assert(status.workspace.name === "ws-js-convenience", "convenience status workspace mismatch");
assert(status.repositories.length === 1, "expected one repository for ws-js-convenience");

({
  ok: true,
  script: "04-convenience-lifecycle",
  discoverCount: discovered.RepositoryCount,
  workspace: created.Workspace.name,
  finalBranch: created.FinalBranch,
  repositoryCount: status.repositories.length,
  overall: status.overall
});
