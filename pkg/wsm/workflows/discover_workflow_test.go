package workflows

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDiscoverPaths(t *testing.T) {
	t.Run("relative path becomes absolute", func(t *testing.T) {
		tmp := t.TempDir()
		orig, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get cwd: %v", err)
		}
		defer func() {
			_ = os.Chdir(orig)
		}()
		if err := os.Chdir(tmp); err != nil {
			t.Fatalf("failed to chdir temp dir: %v", err)
		}

		if err := os.Mkdir("repo", 0o755); err != nil {
			t.Fatalf("failed to create repo directory: %v", err)
		}

		paths, err := ResolveDiscoverPaths([]string{"repo"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(paths) != 1 {
			t.Fatalf("expected 1 path, got %d", len(paths))
		}

		expected := filepath.Join(tmp, "repo")
		if paths[0] != expected {
			t.Fatalf("expected %q, got %q", expected, paths[0])
		}
	})

	t.Run("missing path fails", func(t *testing.T) {
		if _, err := ResolveDiscoverPaths([]string{"/definitely/does/not/exist"}); err == nil {
			t.Fatalf("expected error for missing path")
		}
	})
}
