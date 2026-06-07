package provider

import (
	"encoding/json"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	wsmmodule "github.com/go-go-golems/workspace-manager/pkg/wsmjs/module"
)

func TestRegisterProvider(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, wsmmodule.ModuleName)
	if !ok {
		t.Fatalf("missing module %s.%s", PackageID, wsmmodule.ModuleName)
	}
	if mod.DefaultAs != wsmmodule.ModuleName {
		t.Fatalf("default alias = %q, want %q", mod.DefaultAs, wsmmodule.ModuleName)
	}
}

func TestModuleLoaderInstallsWsmExports(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, wsmmodule.ModuleName)
	if !ok {
		t.Fatalf("missing module")
	}
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{
		Name:   wsmmodule.ModuleName,
		As:     wsmmodule.ModuleName,
		Config: json.RawMessage(`{"defaultJobs": 4}`),
	})
	if err != nil {
		t.Fatalf("create loader: %v", err)
	}

	vm := goja.New()
	moduleObj := vm.NewObject()
	exports := vm.NewObject()
	if err := moduleObj.Set("exports", exports); err != nil {
		t.Fatalf("set exports: %v", err)
	}
	loader(vm, moduleObj)
	if got := exports.Get("version").String(); got != "0.2.0" {
		t.Fatalf("version = %q, want 0.2.0", got)
	}
	if _, ok := goja.AssertFunction(exports.Get("createManager")); !ok {
		t.Fatalf("createManager export is not a function")
	}
}

func TestInvalidConfig(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, wsmmodule.ModuleName)
	if !ok {
		t.Fatalf("missing module")
	}
	if _, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{Config: json.RawMessage(`{"defaultJobs": -1}`)}); err == nil {
		t.Fatalf("expected invalid defaultJobs error")
	}
}
