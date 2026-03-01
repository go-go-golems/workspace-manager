/* eslint-disable @typescript-eslint/no-explicit-any */

declare module "wsm" {
  export interface Consts {
    resolutionMode: {
      CREATE_WORKTREE: string;
      ADD_REPOSITORY: string;
      SYNC: string;
    };
    resolutionStrategy: {
      USE_LOCAL: string;
      TRACK_REMOTE: string;
      CREATE_FROM_BASE: string;
      CREATE_FROM_HEAD: string;
    };
    remoteRefKind: {
      NONE: string;
      REMOTE_TRACKING_BRANCH: string;
    };
    remote: {
      ORIGIN: string;
    };
  }

  export interface ManagerOptions {
    defaultJobs?: number;
  }

  export interface DiscoverInput {
    paths?: string[];
    recursive?: boolean;
    maxDepth?: number;
  }

  export interface CreateWorkspaceInput {
    name: string;
    repos: string[];
    branch?: string;
    branchPrefix?: string;
    baseBranch?: string;
    agentSource?: string;
    dryRun?: boolean;
  }

  export interface StatusInput {
    workspaceName?: string;
    jobs?: number;
  }

  export interface ListRepositoriesInput {
    tags?: string[];
  }

  export interface InfoInput {
    workspaceName?: string;
    field?: string;
  }

  export interface AddRepositoryInput {
    workspaceName?: string;
    repoName: string;
    branch?: string;
    force?: boolean;
  }

  export interface RemoveRepositoryInput {
    workspaceName?: string;
    repoName: string;
    force?: boolean;
    removeFiles?: boolean;
  }

  export interface DeleteWorkspaceInput {
    workspaceName?: string;
    removeFiles?: boolean;
    forceWorktrees?: boolean;
  }

  export interface ForkWorkspaceInput {
    newWorkspaceName: string;
    sourceWorkspaceName?: string;
    branch?: string;
    branchPrefix?: string;
    agentSource?: string;
    dryRun?: boolean;
  }

  export interface MergeWorkspaceInput {
    workspaceName?: string;
    dryRun?: boolean;
    force?: boolean;
    keepWorkspace?: boolean;
  }

  export interface FileChange {
    repository: string;
    file_path: string;
    status: string;
    staged: boolean;
  }

  export interface CommitInput {
    workspaceName?: string;
    message?: string;
    template?: string;
    addAll?: boolean;
    push?: boolean;
    dryRun?: boolean;
    selectedChanges?: Record<string, FileChange[]>;
  }

  export interface DiffInput {
    workspaceName?: string;
    staged?: boolean;
    repo?: string;
    jobs?: number;
  }

  export interface LogInput {
    workspaceName?: string;
    since?: string;
    oneline?: boolean;
    limit?: number;
  }

  export interface BranchCreateInput {
    workspaceName?: string;
    repo?: string;
    branchName: string;
    track?: boolean;
  }

  export interface BranchSwitchInput {
    workspaceName?: string;
    repo?: string;
    branchName: string;
  }

  export interface BranchListInput {
    workspaceName?: string;
    repo?: string;
    jobs?: number;
  }

  export interface RebaseRunInput {
    workspaceName?: string;
    repository?: string;
    targetBranch?: string;
    interactive?: boolean;
    dryRun?: boolean;
    jobs?: number;
    manual?: boolean;
  }

  export interface RebaseStatusInput {
    workspaceName?: string;
    repository?: string;
    jobs?: number;
  }

  export interface RebaseActionInput {
    workspaceName?: string;
    repository?: string;
    jobs?: number;
  }

  export interface RegistryNamespace {
    listRepositories(input?: ListRepositoriesInput): any[];
    listWorkspaces(): any[];
  }

  export interface WorkspacesNamespace {
    create(input: CreateWorkspaceInput): any;
    list(): any[];
    status(input?: StatusInput): any;
    info(input?: InfoInput): any;
    add(input: AddRepositoryInput): any;
    remove(input: RemoveRepositoryInput): any;
    "delete"(input: DeleteWorkspaceInput): any;
    fork(input: ForkWorkspaceInput): any;
    merge(input: MergeWorkspaceInput): any;
  }

  export interface BranchNamespace {
    create(input: BranchCreateInput): any;
    switch(input: BranchSwitchInput): any;
    list(input?: BranchListInput): any;
  }

  export interface RebaseNamespace {
    run(input?: RebaseRunInput): any;
    status(input?: RebaseStatusInput): any;
    "continue"(input?: RebaseActionInput): any;
    abort(input?: RebaseActionInput): any;
  }

  export interface GitNamespace {
    status(input?: StatusInput): any;
    commit(input: CommitInput): any;
    diff(input?: DiffInput): any;
    log(input?: LogInput): any;
    branch: BranchNamespace;
    rebase: RebaseNamespace;
  }

  export interface WorkspaceHandle {
    name(): string;
    path(): string;
    info(input?: Omit<InfoInput, "workspaceName">): any;
    status(input?: Omit<StatusInput, "workspaceName">): any;
    addRepository(input: Omit<AddRepositoryInput, "workspaceName">): any;
    removeRepository(input: Omit<RemoveRepositoryInput, "workspaceName">): any;
    "delete"(input?: Omit<DeleteWorkspaceInput, "workspaceName">): any;
    merge(input?: Omit<MergeWorkspaceInput, "workspaceName">): any;
    git: {
      status(input?: Omit<StatusInput, "workspaceName">): any;
      commit(input: Omit<CommitInput, "workspaceName">): any;
      diff(input?: Omit<DiffInput, "workspaceName">): any;
      log(input?: Omit<LogInput, "workspaceName">): any;
      branch: {
        create(input: Omit<BranchCreateInput, "workspaceName">): any;
        switch(input: Omit<BranchSwitchInput, "workspaceName">): any;
        list(input?: Omit<BranchListInput, "workspaceName">): any;
      };
      rebase: {
        run(input?: Omit<RebaseRunInput, "workspaceName">): any;
        status(input?: Omit<RebaseStatusInput, "workspaceName">): any;
        "continue"(input?: Omit<RebaseActionInput, "workspaceName">): any;
        abort(input?: Omit<RebaseActionInput, "workspaceName">): any;
      };
    };
  }

  export interface Manager {
    discover(input?: DiscoverInput): any;
    createWorkspace(input: CreateWorkspaceInput): any;
    status(input?: StatusInput): any;
    listWorkspaces(): any[];
    listRepositories(input?: ListRepositoriesInput): any[];
    loadWorkspace(name: string): WorkspaceHandle;

    info(input?: InfoInput): any;
    addRepository(input: AddRepositoryInput): any;
    removeRepository(input: RemoveRepositoryInput): any;
    deleteWorkspace(input: DeleteWorkspaceInput): any;
    forkWorkspace(input: ForkWorkspaceInput): any;
    mergeWorkspace(input: MergeWorkspaceInput): any;

    commit(input: CommitInput): any;
    diff(input?: DiffInput): any;
    log(input?: LogInput): any;

    registry: RegistryNamespace;
    workspaces: WorkspacesNamespace;
    git: GitNamespace;
  }

  export interface WSMModule {
    version: string;
    consts: Consts;
    createManager(options?: ManagerOptions): Manager;

    discover(input?: DiscoverInput): any;
    createWorkspace(input: CreateWorkspaceInput): any;
    status(input?: StatusInput): any;
  }

  const wsm: WSMModule;
  export = wsm;
}
