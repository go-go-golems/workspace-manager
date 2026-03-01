package module

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
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

// Register registers the native wsm module on a require registry.
func Register(reg *require.Registry, opts Options) {
	if reg == nil {
		return
	}
	mod := &module{opts: opts}
	reg.RegisterNativeModule(ModuleName, mod.Loader)
}

type module struct {
	opts Options
}

type moduleRuntime struct {
	vm             *goja.Runtime
	managerOptions service.ManagerOptions
}

func (m *module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	rt := &moduleRuntime{vm: vm, managerOptions: m.opts.ManagerOptions}
	exports := moduleObj.Get("exports").(*goja.Object)
	rt.installExports(exports)
}

func (m *moduleRuntime) installExports(exports *goja.Object) {
	m.mustSet(exports, "version", "0.1.0")
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

	registryObj := m.vm.NewObject()
	m.mustSet(registryObj, "listRepositories", listRepositoriesFn)
	m.mustSet(o, "registry", registryObj)

	workspacesObj := m.vm.NewObject()
	m.mustSet(workspacesObj, "create", createWorkspaceFn)
	m.mustSet(workspacesObj, "list", listWorkspacesFn)
	m.mustSet(workspacesObj, "status", statusFn)
	m.mustSet(o, "workspaces", workspacesObj)

	gitObj := m.vm.NewObject()
	m.mustSet(gitObj, "status", statusFn)
	m.mustSet(o, "git", gitObj)

	return o
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
