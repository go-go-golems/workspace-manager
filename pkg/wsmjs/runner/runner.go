package runner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	wsmjsmodule "github.com/go-go-golems/workspace-manager/pkg/wsmjs/module"
)

// RunFile executes a JavaScript file with the wsm native module registered.
func RunFile(ctx context.Context, scriptPath string) (any, error) {
	_ = ctx
	source, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, err
	}
	return RunSource(context.Background(), filepath.Base(scriptPath), string(source))
}

// RunSource executes an in-memory JavaScript source string.
func RunSource(ctx context.Context, sourceName string, source string) (any, error) {
	_ = ctx
	vm := goja.New()
	reg := require.NewRegistry()
	wsmjsmodule.Register(reg, wsmjsmodule.Options{})
	reg.Enable(vm)
	console.Enable(vm)

	v, err := vm.RunScript(sourceName, source)
	if err != nil {
		return nil, err
	}
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil, nil
	}
	return v.Export(), nil
}
