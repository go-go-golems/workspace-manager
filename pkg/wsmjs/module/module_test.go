package module

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
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
