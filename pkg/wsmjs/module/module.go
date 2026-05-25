package module

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	branchsvc "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"github.com/go-go-golems/workspace-manager/pkg/wsmjs/service"
)

const (
	// ModuleName is the JS module identifier used with require("wsm").
	ModuleName = "wsm"
	// hiddenRefKey stores non-enumerable Go pointers on JS objects.
	hiddenRefKey = "__wsm_ref"
)

// Options configures module defaults.
type Options struct {
	ManagerOptions service.ManagerOptions
}

// NewLoader returns the native wsm module loader for use with a require
// registry or an xgoja provider wrapper.
func NewLoader(opts Options) require.ModuleLoader {
	mod := &module{opts: opts}
	return mod.Loader
}

// Register registers the native wsm module on a require registry.
func Register(reg *require.Registry, opts Options) {
	if reg == nil {
		return
	}
	reg.RegisterNativeModule(ModuleName, NewLoader(opts))
}

type module struct {
	opts Options
}

type moduleRuntime struct {
	vm             *goja.Runtime
	managerOptions service.ManagerOptions
}

type workspaceHandleRef struct {
	workspaceName string
}

func (m *module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	rt := &moduleRuntime{vm: vm, managerOptions: m.opts.ManagerOptions}
	exports := moduleObj.Get("exports").(*goja.Object)
	rt.installExports(exports)
}

func (m *moduleRuntime) installExports(exports *goja.Object) {
	m.mustSet(exports, "version", "0.2.0")
	m.mustSet(exports, "consts", m.buildConstsObject())
	m.mustSet(exports, "createManager", m.createManager)
	m.mustSet(exports, "discover", m.discover)
	m.mustSet(exports, "createWorkspace", m.createWorkspace)
	m.mustSet(exports, "status", m.status)
}

func (m *moduleRuntime) createManager(call goja.FunctionCall) goja.Value {
	opts := m.managerOptions
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		input := decodeMap(call.Arguments[0].Export())
		if v, ok := input["defaultJobs"]; ok {
			opts.DefaultJobs = toInt(v, opts.DefaultJobs)
		}
	}

	manager := service.NewManager(opts)
	return m.newManagerObject(manager)
}

func (m *moduleRuntime) discover(call goja.FunctionCall) goja.Value {
	manager := service.NewManager(m.managerOptions)
	input := decodeMapArg(call, 0)

	result, err := manager.Discover(context.Background(), service.DiscoverInput{
		Paths:     toStringSlice(input["paths"]),
		Recursive: toBool(input["recursive"], true),
		MaxDepth:  toInt(input["maxDepth"], 3),
	})
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(result)
}

func (m *moduleRuntime) createWorkspace(call goja.FunctionCall) goja.Value {
	manager := service.NewManager(m.managerOptions)
	input := decodeMapArg(call, 0)

	name := toString(input["name"])
	repos := toStringSlice(input["repos"])
	if name == "" {
		panic(m.vm.NewTypeError("createWorkspace requires name"))
	}
	if len(repos) == 0 {
		panic(m.vm.NewTypeError("createWorkspace requires repos"))
	}

	result, err := manager.CreateWorkspace(context.Background(), service.CreateWorkspaceInput{
		Name:         name,
		Repos:        repos,
		Branch:       toString(input["branch"]),
		BranchPrefix: toString(input["branchPrefix"]),
		BaseBranch:   toString(input["baseBranch"]),
		AgentSource:  toString(input["agentSource"]),
		DryRun:       toBool(input["dryRun"], false),
	})
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(result)
}

func (m *moduleRuntime) status(call goja.FunctionCall) goja.Value {
	manager := service.NewManager(m.managerOptions)
	input := decodeMapArg(call, 0)

	result, err := manager.Status(context.Background(), service.StatusInput{
		WorkspaceName: toString(input["workspaceName"]),
		Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
	})
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.toJSValue(result)
}

func (m *moduleRuntime) newManagerObject(manager *service.Manager) goja.Value {
	o := m.vm.NewObject()
	m.attachRef(o, manager)

	discoverFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.Discover(context.Background(), service.DiscoverInput{
			Paths:     toStringSlice(input["paths"]),
			Recursive: toBool(input["recursive"], true),
			MaxDepth:  toInt(input["maxDepth"], 3),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}
	m.mustSet(o, "discover", discoverFn)

	createWorkspaceFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		name := toString(input["name"])
		repos := toStringSlice(input["repos"])
		if name == "" {
			panic(m.vm.NewTypeError("createWorkspace requires name"))
		}
		if len(repos) == 0 {
			panic(m.vm.NewTypeError("createWorkspace requires repos"))
		}
		result, err := manager.CreateWorkspace(context.Background(), service.CreateWorkspaceInput{
			Name:         name,
			Repos:        repos,
			Branch:       toString(input["branch"]),
			BranchPrefix: toString(input["branchPrefix"]),
			BaseBranch:   toString(input["baseBranch"]),
			AgentSource:  toString(input["agentSource"]),
			DryRun:       toBool(input["dryRun"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}
	m.mustSet(o, "createWorkspace", createWorkspaceFn)

	statusFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.Status(context.Background(), service.StatusInput{
			WorkspaceName: toString(input["workspaceName"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}
	m.mustSet(o, "status", statusFn)

	listWorkspacesFn := func(goja.FunctionCall) goja.Value {
		result, err := manager.ListWorkspaces(context.Background())
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}
	m.mustSet(o, "listWorkspaces", listWorkspacesFn)

	listRepositoriesFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.ListRepositories(context.Background(), service.ListRepositoriesInput{
			Tags: toStringSlice(input["tags"]),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}
	m.mustSet(o, "listRepositories", listRepositoriesFn)

	loadWorkspaceFn := func(call goja.FunctionCall) goja.Value {
		name := ""
		if len(call.Arguments) > 0 {
			if s, ok := call.Arguments[0].Export().(string); ok {
				name = s
			} else {
				input := decodeMapArg(call, 0)
				name = toString(input["name"])
			}
		}
		if name == "" {
			panic(m.vm.NewTypeError("loadWorkspace requires workspace name"))
		}
		workspace, err := manager.LoadWorkspace(context.Background(), name)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.newWorkspaceHandleObject(manager, workspace.Name)
	}
	m.mustSet(o, "loadWorkspace", loadWorkspaceFn)

	infoFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.Info(context.Background(), service.InfoInput{
			WorkspaceName: toString(input["workspaceName"]),
			Field:         toString(input["field"]),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	addFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		if toString(input["workspaceName"]) == "" {
			panic(m.vm.NewTypeError("workspaces.add requires workspaceName"))
		}
		if toString(input["repoName"]) == "" {
			panic(m.vm.NewTypeError("workspaces.add requires repoName"))
		}
		result, err := manager.AddRepository(context.Background(), service.AddRepositoryInput{
			WorkspaceName: toString(input["workspaceName"]),
			RepoName:      toString(input["repoName"]),
			Branch:        toString(input["branch"]),
			Force:         toBool(input["force"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	removeFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		if toString(input["workspaceName"]) == "" {
			panic(m.vm.NewTypeError("workspaces.remove requires workspaceName"))
		}
		if toString(input["repoName"]) == "" {
			panic(m.vm.NewTypeError("workspaces.remove requires repoName"))
		}
		result, err := manager.RemoveRepository(context.Background(), service.RemoveRepositoryInput{
			WorkspaceName: toString(input["workspaceName"]),
			RepoName:      toString(input["repoName"]),
			Force:         toBool(input["force"], false),
			RemoveFiles:   toBool(input["removeFiles"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	deleteFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		if toString(input["workspaceName"]) == "" {
			panic(m.vm.NewTypeError("workspaces.delete requires workspaceName"))
		}
		result, err := manager.DeleteWorkspace(context.Background(), service.DeleteWorkspaceInput{
			WorkspaceName:  toString(input["workspaceName"]),
			RemoveFiles:    toBool(input["removeFiles"], false),
			ForceWorktrees: toBool(input["forceWorktrees"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	forkFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		if toString(input["newWorkspaceName"]) == "" {
			panic(m.vm.NewTypeError("workspaces.fork requires newWorkspaceName"))
		}
		result, err := manager.ForkWorkspace(context.Background(), service.ForkWorkspaceInput{
			NewWorkspaceName:    toString(input["newWorkspaceName"]),
			SourceWorkspaceName: toString(input["sourceWorkspaceName"]),
			Branch:              toString(input["branch"]),
			BranchPrefix:        toString(input["branchPrefix"]),
			AgentSource:         toString(input["agentSource"]),
			DryRun:              toBool(input["dryRun"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	mergeFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		if toString(input["workspaceName"]) == "" {
			panic(m.vm.NewTypeError("workspaces.merge requires workspaceName"))
		}
		result, err := manager.MergeWorkspace(context.Background(), service.MergeWorkspaceInput{
			WorkspaceName: toString(input["workspaceName"]),
			DryRun:        toBool(input["dryRun"], false),
			Force:         toBool(input["force"], false),
			KeepWorkspace: toBool(input["keepWorkspace"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	commitFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		if toString(input["message"]) == "" && toString(input["template"]) == "" {
			panic(m.vm.NewTypeError("git.commit requires message or template"))
		}
		result, err := manager.Commit(context.Background(), service.CommitInput{
			WorkspaceName:   toString(input["workspaceName"]),
			Message:         toString(input["message"]),
			Template:        toString(input["template"]),
			AddAll:          toBool(input["addAll"], false),
			Push:            toBool(input["push"], false),
			DryRun:          toBool(input["dryRun"], false),
			SelectedChanges: decodeSelectedChanges(input["selectedChanges"]),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	diffFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.Diff(context.Background(), service.DiffInput{
			WorkspaceName: toString(input["workspaceName"]),
			Staged:        toBool(input["staged"], false),
			Repo:          toString(input["repo"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	logFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.Log(context.Background(), service.LogInput{
			WorkspaceName: toString(input["workspaceName"]),
			Since:         toString(input["since"]),
			Oneline:       toBool(input["oneline"], false),
			Limit:         toInt(input["limit"], 10),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	branchCreateFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		if toString(input["branchName"]) == "" {
			panic(m.vm.NewTypeError("git.branch.create requires branchName"))
		}
		result, err := manager.BranchCreate(context.Background(), service.BranchCreateInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repo:          toString(input["repo"]),
			BranchName:    toString(input["branchName"]),
			Track:         toBool(input["track"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	branchSwitchFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		if toString(input["branchName"]) == "" {
			panic(m.vm.NewTypeError("git.branch.switch requires branchName"))
		}
		result, err := manager.BranchSwitch(context.Background(), service.BranchSwitchInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repo:          toString(input["repo"]),
			BranchName:    toString(input["branchName"]),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	branchListFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.BranchList(context.Background(), service.BranchListInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repo:          toString(input["repo"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	rebaseRunFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.RebaseRun(context.Background(), service.RebaseRunInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repository:    toString(input["repository"]),
			TargetBranch:  toString(input["targetBranch"]),
			Interactive:   toBool(input["interactive"], false),
			DryRun:        toBool(input["dryRun"], false),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
			Manual:        toBool(input["manual"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	rebaseStatusFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.RebaseStatus(context.Background(), service.RebaseStatusInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repository:    toString(input["repository"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	rebaseContinueFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.RebaseContinue(context.Background(), service.RebaseActionInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repository:    toString(input["repository"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	rebaseAbortFn := func(call goja.FunctionCall) goja.Value {
		input := decodeMapArg(call, 0)
		result, err := manager.RebaseAbort(context.Background(), service.RebaseActionInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repository:    toString(input["repository"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	registryObj := m.vm.NewObject()
	m.mustSet(registryObj, "listRepositories", listRepositoriesFn)
	m.mustSet(registryObj, "listWorkspaces", listWorkspacesFn)
	m.mustSet(o, "registry", registryObj)

	workspacesObj := m.vm.NewObject()
	m.mustSet(workspacesObj, "create", createWorkspaceFn)
	m.mustSet(workspacesObj, "list", listWorkspacesFn)
	m.mustSet(workspacesObj, "status", statusFn)
	m.mustSet(workspacesObj, "info", infoFn)
	m.mustSet(workspacesObj, "add", addFn)
	m.mustSet(workspacesObj, "remove", removeFn)
	m.mustSet(workspacesObj, "delete", deleteFn)
	m.mustSet(workspacesObj, "fork", forkFn)
	m.mustSet(workspacesObj, "merge", mergeFn)
	m.mustSet(o, "workspaces", workspacesObj)

	branchObj := m.vm.NewObject()
	m.mustSet(branchObj, "create", branchCreateFn)
	m.mustSet(branchObj, "switch", branchSwitchFn)
	m.mustSet(branchObj, "list", branchListFn)

	rebaseObj := m.vm.NewObject()
	m.mustSet(rebaseObj, "run", rebaseRunFn)
	m.mustSet(rebaseObj, "status", rebaseStatusFn)
	m.mustSet(rebaseObj, "continue", rebaseContinueFn)
	m.mustSet(rebaseObj, "abort", rebaseAbortFn)

	gitObj := m.vm.NewObject()
	m.mustSet(gitObj, "status", statusFn)
	m.mustSet(gitObj, "commit", commitFn)
	m.mustSet(gitObj, "diff", diffFn)
	m.mustSet(gitObj, "log", logFn)
	m.mustSet(gitObj, "branch", branchObj)
	m.mustSet(gitObj, "rebase", rebaseObj)
	m.mustSet(o, "git", gitObj)

	m.mustSet(o, "info", infoFn)
	m.mustSet(o, "addRepository", addFn)
	m.mustSet(o, "removeRepository", removeFn)
	m.mustSet(o, "deleteWorkspace", deleteFn)
	m.mustSet(o, "forkWorkspace", forkFn)
	m.mustSet(o, "mergeWorkspace", mergeFn)
	m.mustSet(o, "commit", commitFn)
	m.mustSet(o, "diff", diffFn)
	m.mustSet(o, "log", logFn)
	m.mustSet(o, "loadWorkspace", loadWorkspaceFn)

	return o
}

func (m *moduleRuntime) newWorkspaceHandleObject(manager *service.Manager, workspaceName string) goja.Value {
	o := m.vm.NewObject()
	m.attachRef(o, &workspaceHandleRef{workspaceName: workspaceName})

	nameFn := func(goja.FunctionCall) goja.Value {
		return m.vm.ToValue(workspaceName)
	}
	pathFn := func(goja.FunctionCall) goja.Value {
		workspace, err := manager.LoadWorkspace(context.Background(), workspaceName)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.vm.ToValue(workspace.Path)
	}

	infoFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.Info(context.Background(), service.InfoInput{
			WorkspaceName: toString(input["workspaceName"]),
			Field:         toString(input["field"]),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	statusFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.Status(context.Background(), service.StatusInput{
			WorkspaceName: toString(input["workspaceName"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	addFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		if toString(input["repoName"]) == "" {
			panic(m.vm.NewTypeError("workspaceHandle.addRepository requires repoName"))
		}
		result, err := manager.AddRepository(context.Background(), service.AddRepositoryInput{
			WorkspaceName: toString(input["workspaceName"]),
			RepoName:      toString(input["repoName"]),
			Branch:        toString(input["branch"]),
			Force:         toBool(input["force"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	removeFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		if toString(input["repoName"]) == "" {
			panic(m.vm.NewTypeError("workspaceHandle.removeRepository requires repoName"))
		}
		result, err := manager.RemoveRepository(context.Background(), service.RemoveRepositoryInput{
			WorkspaceName: toString(input["workspaceName"]),
			RepoName:      toString(input["repoName"]),
			Force:         toBool(input["force"], false),
			RemoveFiles:   toBool(input["removeFiles"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	deleteFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.DeleteWorkspace(context.Background(), service.DeleteWorkspaceInput{
			WorkspaceName:  toString(input["workspaceName"]),
			RemoveFiles:    toBool(input["removeFiles"], false),
			ForceWorktrees: toBool(input["forceWorktrees"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	mergeFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.MergeWorkspace(context.Background(), service.MergeWorkspaceInput{
			WorkspaceName: toString(input["workspaceName"]),
			DryRun:        toBool(input["dryRun"], false),
			Force:         toBool(input["force"], false),
			KeepWorkspace: toBool(input["keepWorkspace"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	commitFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		if toString(input["message"]) == "" && toString(input["template"]) == "" {
			panic(m.vm.NewTypeError("git.commit requires message or template"))
		}
		result, err := manager.Commit(context.Background(), service.CommitInput{
			WorkspaceName:   toString(input["workspaceName"]),
			Message:         toString(input["message"]),
			Template:        toString(input["template"]),
			AddAll:          toBool(input["addAll"], false),
			Push:            toBool(input["push"], false),
			DryRun:          toBool(input["dryRun"], false),
			SelectedChanges: decodeSelectedChanges(input["selectedChanges"]),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	diffFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.Diff(context.Background(), service.DiffInput{
			WorkspaceName: toString(input["workspaceName"]),
			Staged:        toBool(input["staged"], false),
			Repo:          toString(input["repo"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	logFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.Log(context.Background(), service.LogInput{
			WorkspaceName: toString(input["workspaceName"]),
			Since:         toString(input["since"]),
			Oneline:       toBool(input["oneline"], false),
			Limit:         toInt(input["limit"], 10),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	branchCreateFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		if toString(input["branchName"]) == "" {
			panic(m.vm.NewTypeError("git.branch.create requires branchName"))
		}
		result, err := manager.BranchCreate(context.Background(), service.BranchCreateInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repo:          toString(input["repo"]),
			BranchName:    toString(input["branchName"]),
			Track:         toBool(input["track"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	branchSwitchFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		if toString(input["branchName"]) == "" {
			panic(m.vm.NewTypeError("git.branch.switch requires branchName"))
		}
		result, err := manager.BranchSwitch(context.Background(), service.BranchSwitchInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repo:          toString(input["repo"]),
			BranchName:    toString(input["branchName"]),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	branchListFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.BranchList(context.Background(), service.BranchListInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repo:          toString(input["repo"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	rebaseRunFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.RebaseRun(context.Background(), service.RebaseRunInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repository:    toString(input["repository"]),
			TargetBranch:  toString(input["targetBranch"]),
			Interactive:   toBool(input["interactive"], false),
			DryRun:        toBool(input["dryRun"], false),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
			Manual:        toBool(input["manual"], false),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	rebaseStatusFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.RebaseStatus(context.Background(), service.RebaseStatusInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repository:    toString(input["repository"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	rebaseContinueFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.RebaseContinue(context.Background(), service.RebaseActionInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repository:    toString(input["repository"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	rebaseAbortFn := func(call goja.FunctionCall) goja.Value {
		input := m.withWorkspaceName(decodeMapArg(call, 0), workspaceName)
		result, err := manager.RebaseAbort(context.Background(), service.RebaseActionInput{
			WorkspaceName: toString(input["workspaceName"]),
			Repository:    toString(input["repository"]),
			Jobs:          toInt(input["jobs"], m.managerOptions.DefaultJobs),
		})
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.toJSValue(result)
	}

	branchObj := m.vm.NewObject()
	m.mustSet(branchObj, "create", branchCreateFn)
	m.mustSet(branchObj, "switch", branchSwitchFn)
	m.mustSet(branchObj, "list", branchListFn)

	rebaseObj := m.vm.NewObject()
	m.mustSet(rebaseObj, "run", rebaseRunFn)
	m.mustSet(rebaseObj, "status", rebaseStatusFn)
	m.mustSet(rebaseObj, "continue", rebaseContinueFn)
	m.mustSet(rebaseObj, "abort", rebaseAbortFn)

	gitObj := m.vm.NewObject()
	m.mustSet(gitObj, "status", statusFn)
	m.mustSet(gitObj, "commit", commitFn)
	m.mustSet(gitObj, "diff", diffFn)
	m.mustSet(gitObj, "log", logFn)
	m.mustSet(gitObj, "branch", branchObj)
	m.mustSet(gitObj, "rebase", rebaseObj)

	m.mustSet(o, "name", nameFn)
	m.mustSet(o, "path", pathFn)
	m.mustSet(o, "info", infoFn)
	m.mustSet(o, "status", statusFn)
	m.mustSet(o, "addRepository", addFn)
	m.mustSet(o, "removeRepository", removeFn)
	m.mustSet(o, "delete", deleteFn)
	m.mustSet(o, "merge", mergeFn)
	m.mustSet(o, "git", gitObj)

	return o
}

func (m *moduleRuntime) withWorkspaceName(input map[string]any, workspaceName string) map[string]any {
	if input == nil {
		input = map[string]any{}
	}
	result := map[string]any{}
	for k, v := range input {
		result[k] = v
	}
	if toString(result["workspaceName"]) == "" {
		result["workspaceName"] = workspaceName
	}
	return result
}

func (m *moduleRuntime) buildConstsObject() *goja.Object {
	constsObj := m.vm.NewObject()

	resolutionModeObj := m.vm.NewObject()
	m.mustSet(resolutionModeObj, "CREATE_WORKTREE", branchsvc.ResolutionModeCreateWorktree.String())
	m.mustSet(resolutionModeObj, "ADD_REPOSITORY", branchsvc.ResolutionModeAddRepository.String())
	m.mustSet(resolutionModeObj, "SYNC", branchsvc.ResolutionModeSync.String())
	m.mustSet(constsObj, "resolutionMode", resolutionModeObj)

	resolutionStrategyObj := m.vm.NewObject()
	m.mustSet(resolutionStrategyObj, "USE_LOCAL", branchsvc.ResolutionStrategyUseLocal.String())
	m.mustSet(resolutionStrategyObj, "TRACK_REMOTE", branchsvc.ResolutionStrategyTrackRemote.String())
	m.mustSet(resolutionStrategyObj, "CREATE_FROM_BASE", branchsvc.ResolutionStrategyCreateFromBase.String())
	m.mustSet(resolutionStrategyObj, "CREATE_FROM_HEAD", branchsvc.ResolutionStrategyCreateFromHead.String())
	m.mustSet(constsObj, "resolutionStrategy", resolutionStrategyObj)

	remoteRefKindObj := m.vm.NewObject()
	m.mustSet(remoteRefKindObj, "NONE", branchsvc.RemoteRefKindNone.String())
	m.mustSet(remoteRefKindObj, "REMOTE_TRACKING_BRANCH", branchsvc.RemoteRefKindRemoteTrackingBranch.String())
	m.mustSet(constsObj, "remoteRefKind", remoteRefKindObj)

	remoteObj := m.vm.NewObject()
	m.mustSet(remoteObj, "ORIGIN", string(branchsvc.DefaultRemoteName))
	m.mustSet(constsObj, "remote", remoteObj)

	return constsObj
}

func (m *moduleRuntime) mustSet(o *goja.Object, key string, v any) {
	if err := o.Set(key, v); err != nil {
		panic(m.vm.NewGoError(fmt.Errorf("set %s: %w", key, err)))
	}
}

func (m *moduleRuntime) attachRef(o *goja.Object, ref any) {
	_ = o.Set(hiddenRefKey, ref)
	_ = o.DefineDataProperty(hiddenRefKey, o.Get(hiddenRefKey),
		goja.FLAG_FALSE,
		goja.FLAG_FALSE,
		goja.FLAG_FALSE,
	)
}

func (m *moduleRuntime) toJSValue(v any) goja.Value {
	if v == nil {
		return goja.Null()
	}
	encoded, err := toPortableValue(v)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.vm.ToValue(encoded)
}

func toPortableValue(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeMapArg(call goja.FunctionCall, idx int) map[string]any {
	if len(call.Arguments) <= idx {
		return map[string]any{}
	}
	v := call.Arguments[idx]
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return map[string]any{}
	}
	return decodeMap(v.Export())
}

func decodeMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func decodeSelectedChanges(v any) map[string][]wsm.FileChange {
	if v == nil {
		return nil
	}
	if out, ok := v.(map[string][]wsm.FileChange); ok {
		return out
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	out := map[string][]wsm.FileChange{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}

func toString(v any) string {
	s, ok := v.(string)
	if ok {
		return s
	}
	return ""
}

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	if values, ok := v.([]string); ok {
		return values
	}
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	ret := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			ret = append(ret, s)
		}
	}
	return ret
}

func toBool(v any, fallback bool) bool {
	if v == nil {
		return fallback
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return fallback
}

func toInt(v any, fallback int) int {
	if v == nil {
		return fallback
	}
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case float32:
		return int(x)
	default:
		return fallback
	}
}
