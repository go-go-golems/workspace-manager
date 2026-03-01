package helpers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Sandbox provides an isolated HOME and filesystem area for integration tests.
type Sandbox struct {
	T          *testing.T
	BaseDir    string
	HomeDir    string
	ReposDir   string
	RemotesDir string
	defaultEnv map[string]string
}

// NewSandbox creates a new temporary sandbox. Caller should defer s.Cleanup().
func NewSandbox(t *testing.T) *Sandbox {
	t.Helper()
	base := t.TempDir()
	home := filepath.Join(base, "home")
	repos := filepath.Join(base, "repos")
	remotes := filepath.Join(base, "remotes")
	must(os.MkdirAll(home, 0o755))
	must(os.MkdirAll(repos, 0o755))
	must(os.MkdirAll(remotes, 0o755))
	must(os.MkdirAll(filepath.Join(home, ".config"), 0o755))
	must(os.MkdirAll(filepath.Join(home, ".cache"), 0o755))
	must(os.MkdirAll(filepath.Join(home, ".local", "state"), 0o755))

	hostGoModCache := os.Getenv("GOMODCACHE")
	if hostGoModCache == "" {
		if hostHome, err := os.UserHomeDir(); err == nil {
			hostGoModCache = filepath.Join(hostHome, "go", "pkg", "mod")
		}
	}
	hostGoBuildCache := os.Getenv("GOCACHE")
	if hostGoBuildCache == "" {
		if cacheDir, err := os.UserCacheDir(); err == nil {
			hostGoBuildCache = filepath.Join(cacheDir, "go-build")
		}
	}

	// Baseline default env for subprocesses (WSM and git)
	env := map[string]string{
		"HOME": home,
		// Force config/cache/state into sandbox to avoid leaking host machine settings.
		"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
		"XDG_CACHE_HOME":  filepath.Join(home, ".cache"),
		"XDG_STATE_HOME":  filepath.Join(home, ".local", "state"),
		// Ensure deterministic locale to make output assertions stable
		"LC_ALL": "C",
		"LANG":   "C",
		// Provide a default git identity for commits
		"GIT_AUTHOR_NAME":     "WSM Test",
		"GIT_AUTHOR_EMAIL":    "wsm@example.com",
		"GIT_COMMITTER_NAME":  "WSM Test",
		"GIT_COMMITTER_EMAIL": "wsm@example.com",
		// Force non-interactive git flows for rebase/cherry-pick style operations.
		"GIT_EDITOR":          "true",
		"GIT_SEQUENCE_EDITOR": "true",
	}
	if hostGoModCache != "" {
		env["GOMODCACHE"] = hostGoModCache
	}
	if hostGoBuildCache != "" {
		env["GOCACHE"] = hostGoBuildCache
	}

	sb := &Sandbox{T: t, BaseDir: base, HomeDir: home, ReposDir: repos, RemotesDir: remotes, defaultEnv: env}
	sb.DebugFS(t)
	return sb
}

// Cleanup performs any additional cleanup if needed.
func (s *Sandbox) Cleanup() {
	// Nothing custom beyond t.TempDir cleanup for now.
}

// TempDir returns a path under the sandbox base for ad-hoc usage.
func (s *Sandbox) TempDir(parts ...string) string {
	p := filepath.Join(append([]string{s.BaseDir}, parts...)...)
	must(os.MkdirAll(p, 0o755))
	return p
}

// Env returns the base environment variables array for subprocesses.
func (s *Sandbox) Env(extra map[string]string) []string {
	merged := map[string]string{}
	for k, v := range s.defaultEnv {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	// Convert to ["K=V", ...]
	// Start from current process env, but override with merged to avoid missing basics like PATH.
	out := append([]string{}, os.Environ()...)
	for k, v := range merged {
		out = upsertEnv(out, k, v)
	}
	return out
}

// LoadWorkspacePath reads the workspace path from the config JSON persisted by WSM.
func (s *Sandbox) LoadWorkspacePath(t *testing.T, name string) string {
	t.Helper()
	// Respect sandbox HOME by deriving from HOME
	cfgDir := filepath.Join(s.HomeDir, ".config")
	wsFile := filepath.Join(cfgDir, "workspace-manager", "workspaces", name+".json")
	data, err := os.ReadFile(wsFile)
	if err != nil {
		t.Fatalf("read workspace file: %v", err)
	}
	var ws struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(data, &ws); err != nil {
		t.Fatalf("parse workspace file: %v", err)
	}
	return ws.Path
}

// Context returns a cancellable context for subprocess timeouts if needed.
func (s *Sandbox) Context(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}

// DebugFS logs useful information about the current working directory and common paths.
func (s *Sandbox) DebugFS(t *testing.T) {
	t.Helper()
	cwd, _ := os.Getwd()
	t.Logf("[debug] CWD: %s", cwd)
	t.Logf("[debug] Sandbox Base: %s | Home: %s | Repos: %s | Remotes: %s", s.BaseDir, s.HomeDir, s.ReposDir, s.RemotesDir)

	// Helper to list a directory (non-recursive)
	list := func(path string) {
		entries, err := os.ReadDir(path)
		if err != nil {
			t.Logf("[debug] ls %s: %v", path, err)
			return
		}
		maxEntries := 20
		for i, e := range entries {
			if i >= maxEntries {
				t.Logf("[debug] ... (%d more)", len(entries)-maxEntries)
				break
			}
			info, _ := e.Info()
			t.Logf("[debug] %s %10d %s", map[bool]string{true: "d", false: "-"}[e.IsDir()], info.Size(), filepath.Join(path, e.Name()))
		}
	}

	for _, p := range []string{".", "test", filepath.Join("test", "integration"), "cmd", "pkg"} {
		list(p)
	}

	// Count test files
	count := 0
	_ = filepath.WalkDir("test", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && filepath.Ext(path) == ".go" && len(path) >= 8 && path[len(path)-8:] == "_test.go" {
			count++
		}
		return nil
	})
	t.Logf("[debug] Found %d *_test.go files under ./test", count)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func upsertEnv(env []string, key, val string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			env[i] = prefix + val
			return env
		}
	}
	return append(env, prefix+val)
}
