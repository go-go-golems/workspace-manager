package module

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/workspace-manager/pkg/wsmjs/service"
)

func TestModuleExportsCoreSurface(t *testing.T) {
	vm := goja.New()
	reg := require.NewRegistry()
	Register(reg, Options{})
	req := reg.Enable(vm)
	if req == nil {
		t.Fatalf("expected require module")
	}

	script := `
const wsm = require("wsm");
if (!wsm || typeof wsm.createManager !== "function") {
  throw new Error("missing createManager");
}
if (!wsm.consts || wsm.consts.remote.ORIGIN !== "origin") {
  throw new Error("missing consts.remote.ORIGIN");
}
if (wsm.consts.resolutionMode.CREATE_WORKTREE !== "create-worktree") {
  throw new Error("missing CREATE_WORKTREE const");
}
const manager = wsm.createManager({ defaultJobs: 5 });
({
  ok: true,
  hasDiscover: typeof manager.discover === "function",
  hasCreateWorkspace: typeof manager.createWorkspace === "function",
  hasStatus: typeof manager.status === "function",
  hasRegistryList: typeof manager.registry.listRepositories === "function",
  hasGitStatus: typeof manager.git.status === "function",
  hasWorkspacesList: typeof manager.workspaces.list === "function"
});
`

	v, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("run script: %v", err)
	}
	obj := v.Export().(map[string]any)
	if ok, _ := obj["ok"].(bool); !ok {
		t.Fatalf("expected ok=true")
	}
	if hasDiscover, _ := obj["hasDiscover"].(bool); !hasDiscover {
		t.Fatalf("expected manager.discover")
	}
	if hasCreateWorkspace, _ := obj["hasCreateWorkspace"].(bool); !hasCreateWorkspace {
		t.Fatalf("expected manager.createWorkspace")
	}
	if hasStatus, _ := obj["hasStatus"].(bool); !hasStatus {
		t.Fatalf("expected manager.status")
	}
	if hasRegistryList, _ := obj["hasRegistryList"].(bool); !hasRegistryList {
		t.Fatalf("expected manager.registry.listRepositories")
	}
	if hasGitStatus, _ := obj["hasGitStatus"].(bool); !hasGitStatus {
		t.Fatalf("expected manager.git.status")
	}
	if hasWorkspacesList, _ := obj["hasWorkspacesList"].(bool); !hasWorkspacesList {
		t.Fatalf("expected manager.workspaces.list")
	}
}

func TestModuleExportsExpandedSurface(t *testing.T) {
	vm := goja.New()
	reg := require.NewRegistry()
	Register(reg, Options{})
	req := reg.Enable(vm)
	if req == nil {
		t.Fatalf("expected require module")
	}

	script := `
const wsm = require("wsm");
const manager = wsm.createManager({ defaultJobs: 6 });
({
  hasLoadWorkspace: typeof manager.loadWorkspace === "function",
  hasRegistryListWorkspaces: typeof manager.registry.listWorkspaces === "function",
  hasWorkspacesInfo: typeof manager.workspaces.info === "function",
  hasWorkspacesAdd: typeof manager.workspaces.add === "function",
  hasWorkspacesRemove: typeof manager.workspaces.remove === "function",
  hasWorkspacesDelete: typeof manager.workspaces.delete === "function",
  hasWorkspacesFork: typeof manager.workspaces.fork === "function",
  hasWorkspacesMerge: typeof manager.workspaces.merge === "function",
  hasGitCommit: typeof manager.git.commit === "function",
  hasGitDiff: typeof manager.git.diff === "function",
  hasGitLog: typeof manager.git.log === "function",
  hasBranchCreate: typeof manager.git.branch.create === "function",
  hasBranchSwitch: typeof manager.git.branch.switch === "function",
  hasBranchList: typeof manager.git.branch.list === "function",
  hasRebaseRun: typeof manager.git.rebase.run === "function",
  hasRebaseStatus: typeof manager.git.rebase.status === "function",
  hasRebaseContinue: typeof manager.git.rebase.continue === "function",
  hasRebaseAbort: typeof manager.git.rebase.abort === "function",
  hasFlatCommit: typeof manager.commit === "function",
  hasFlatDiff: typeof manager.diff === "function",
  hasFlatLog: typeof manager.log === "function",
  hasFlatAdd: typeof manager.addRepository === "function",
  hasFlatRemove: typeof manager.removeRepository === "function",
  hasFlatDelete: typeof manager.deleteWorkspace === "function",
  hasFlatFork: typeof manager.forkWorkspace === "function",
  hasFlatMerge: typeof manager.mergeWorkspace === "function"
});
`

	v, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("run script: %v", err)
	}
	obj := v.Export().(map[string]any)
	for k, v := range obj {
		ok, _ := v.(bool)
		if !ok {
			t.Fatalf("expected %s=true, got %#v", k, obj[k])
		}
	}
}

func TestModuleExpandedValidationErrors(t *testing.T) {
	vm := goja.New()
	reg := require.NewRegistry()
	Register(reg, Options{})
	req := reg.Enable(vm)
	if req == nil {
		t.Fatalf("expected require module")
	}

	script := `
const wsm = require("wsm");
const manager = wsm.createManager();

const capture = (fn) => {
  try {
    fn();
    return "NO_ERROR";
  } catch (err) {
    return String(err && err.message ? err.message : err);
  }
};

({
  loadWorkspace: capture(() => manager.loadWorkspace()),
  add: capture(() => manager.workspaces.add({})),
  remove: capture(() => manager.workspaces.remove({ workspaceName: "ws" })),
  delete: capture(() => manager.workspaces.delete({})),
  fork: capture(() => manager.workspaces.fork({})),
  merge: capture(() => manager.workspaces.merge({})),
  commit: capture(() => manager.git.commit({})),
  branchCreate: capture(() => manager.git.branch.create({})),
  branchSwitch: capture(() => manager.git.branch.switch({}))
});
`

	v, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("run script: %v", err)
	}
	obj := v.Export().(map[string]any)

	assertContainsMessage(t, obj["loadWorkspace"], "loadWorkspace requires workspace name")
	assertContainsMessage(t, obj["add"], "workspaces.add requires workspaceName")
	assertContainsMessage(t, obj["remove"], "workspaces.remove requires repoName")
	assertContainsMessage(t, obj["delete"], "workspaces.delete requires workspaceName")
	assertContainsMessage(t, obj["fork"], "workspaces.fork requires newWorkspaceName")
	assertContainsMessage(t, obj["merge"], "workspaces.merge requires workspaceName")
	assertContainsMessage(t, obj["commit"], "git.commit requires message or template")
	assertContainsMessage(t, obj["branchCreate"], "git.branch.create requires branchName")
	assertContainsMessage(t, obj["branchSwitch"], "git.branch.switch requires branchName")
}

func TestWorkspaceHandleSurfaceAndScopeInjection(t *testing.T) {
	vm := goja.New()
	rt := &moduleRuntime{
		vm:             vm,
		managerOptions: service.ManagerOptions{DefaultJobs: 4},
	}
	manager := service.NewManager(service.ManagerOptions{DefaultJobs: 4})

	value := rt.newWorkspaceHandleObject(manager, "ws-handle")
	obj := value.ToObject(vm)
	if obj == nil {
		t.Fatalf("expected workspace handle object")
	}

	surface := []string{
		"name", "path", "info", "status",
		"addRepository", "removeRepository", "delete", "merge", "git",
	}
	for _, key := range surface {
		got := obj.Get(key)
		if goja.IsUndefined(got) || goja.IsNull(got) {
			t.Fatalf("missing workspace handle key: %s", key)
		}
	}

	injected := rt.withWorkspaceName(map[string]any{"jobs": 3}, "ws-handle")
	if injected["workspaceName"] != "ws-handle" {
		t.Fatalf("expected injected workspaceName=ws-handle, got %#v", injected)
	}

	preserved := rt.withWorkspaceName(map[string]any{
		"workspaceName": "ws-explicit",
		"jobs":          5,
	}, "ws-handle")
	if preserved["workspaceName"] != "ws-explicit" {
		t.Fatalf("expected explicit workspaceName preserved, got %#v", preserved)
	}
}

func assertContainsMessage(t *testing.T, value any, want string) {
	t.Helper()
	got, _ := value.(string)
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q", got, want)
	}
}
