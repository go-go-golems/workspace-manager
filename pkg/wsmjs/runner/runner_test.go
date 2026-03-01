package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunFileWithWSMModule(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "smoke.js")
	script := `
const wsm = require("wsm");
({
  ok: true,
  origin: wsm.consts.remote.ORIGIN,
  hasCreateManager: typeof wsm.createManager === "function"
});
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}

	result, err := RunFile(context.Background(), scriptPath)
	if err != nil {
		t.Fatalf("run file: %v", err)
	}

	obj, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if okVal, _ := obj["ok"].(bool); !okVal {
		t.Fatalf("expected ok=true")
	}
	if origin, _ := obj["origin"].(string); origin != "origin" {
		t.Fatalf("expected origin=origin, got %q", origin)
	}
	if hasCreateManager, _ := obj["hasCreateManager"].(bool); !hasCreateManager {
		t.Fatalf("expected hasCreateManager=true")
	}
}
