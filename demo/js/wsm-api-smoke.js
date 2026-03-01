const wsm = require("wsm");

const manager = wsm.createManager({ defaultJobs: 8 });

({
  moduleVersion: wsm.version,
  consts: {
    remote: wsm.consts.remote.ORIGIN,
    resolutionModeCreateWorktree: wsm.consts.resolutionMode.CREATE_WORKTREE,
    resolutionStrategyUseLocal: wsm.consts.resolutionStrategy.USE_LOCAL,
  },
  managerSurface: {
    discover: typeof manager.discover,
    createWorkspace: typeof manager.createWorkspace,
    status: typeof manager.status,
    listWorkspaces: typeof manager.listWorkspaces,
    listRepositories: typeof manager.listRepositories,
    registryListRepositories: typeof manager.registry.listRepositories,
    workspacesList: typeof manager.workspaces.list,
    gitStatus: typeof manager.git.status,
  },
});
