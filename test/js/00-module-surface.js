var wsm = require("wsm");

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

var manager = wsm.createManager({ defaultJobs: 4 });

assert(typeof wsm.version === "string" && wsm.version.length > 0, "wsm.version must be a non-empty string");
assert(wsm.consts.remote.ORIGIN === "origin", "wsm.consts.remote.ORIGIN must equal origin");
assert(wsm.consts.resolutionMode.CREATE_WORKTREE === "create-worktree", "expected create-worktree resolution mode");
assert(typeof manager.discover === "function", "manager.discover missing");
assert(typeof manager.createWorkspace === "function", "manager.createWorkspace missing");
assert(typeof manager.status === "function", "manager.status missing");
assert(typeof manager.listWorkspaces === "function", "manager.listWorkspaces missing");
assert(typeof manager.listRepositories === "function", "manager.listRepositories missing");
assert(typeof manager.registry.listRepositories === "function", "manager.registry.listRepositories missing");
assert(typeof manager.workspaces.create === "function", "manager.workspaces.create missing");
assert(typeof manager.workspaces.list === "function", "manager.workspaces.list missing");
assert(typeof manager.workspaces.status === "function", "manager.workspaces.status missing");
assert(typeof manager.git.status === "function", "manager.git.status missing");

({
  ok: true,
  script: "00-module-surface",
  version: wsm.version,
  consts: {
    origin: wsm.consts.remote.ORIGIN,
    modeCreateWorktree: wsm.consts.resolutionMode.CREATE_WORKTREE,
    strategyUseLocal: wsm.consts.resolutionStrategy.USE_LOCAL,
    remoteRefNone: wsm.consts.remoteRefKind.NONE
  },
  managerSurface: {
    discover: typeof manager.discover,
    createWorkspace: typeof manager.createWorkspace,
    status: typeof manager.status,
    listWorkspaces: typeof manager.listWorkspaces,
    listRepositories: typeof manager.listRepositories,
    registryListRepositories: typeof manager.registry.listRepositories,
    workspacesCreate: typeof manager.workspaces.create,
    workspacesList: typeof manager.workspaces.list,
    workspacesStatus: typeof manager.workspaces.status,
    gitStatus: typeof manager.git.status
  }
});
