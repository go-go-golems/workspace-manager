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
    T             *testing.T
    BaseDir       string
    HomeDir       string
    ReposDir      string
    RemotesDir    string
    defaultEnv    map[string]string
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

    // Baseline default env for subprocesses (WSM and git)
    env := map[string]string{
        "HOME": home,
        // Ensure deterministic locale to make output assertions stable
        "LC_ALL": "C",
        "LANG": "C",
        // Provide a default git identity for commits
        "GIT_AUTHOR_NAME": "WSM Test",
        "GIT_AUTHOR_EMAIL": "wsm@example.com",
        "GIT_COMMITTER_NAME": "WSM Test",
        "GIT_COMMITTER_EMAIL": "wsm@example.com",
    }

    return &Sandbox{T: t, BaseDir: base, HomeDir: home, ReposDir: repos, RemotesDir: remotes, defaultEnv: env}
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
    for k, v := range s.defaultEnv { merged[k] = v }
    for k, v := range extra { merged[k] = v }
    // Convert to ["K=V", ...]
    var out []string
    // Start from current process env, but override with merged to avoid missing basics like PATH
    for _, e := range os.Environ() { out = append(out, e) }
    for k, v := range merged { out = upsertEnv(out, k, v) }
    return out
}

// SetBackend sets default backend env for WSM subprocesses.
func (s *Sandbox) SetBackend(backend string) {
    s.defaultEnv["WSM_GIT_BACKEND"] = backend
}

// LoadWorkspacePath reads the workspace path from the config JSON persisted by WSM.
func (s *Sandbox) LoadWorkspacePath(t *testing.T, name string) string {
    t.Helper()
    cfgDir, err := os.UserConfigDir()
    if err != nil { t.Fatalf("config dir: %v", err) }
    // Respect sandbox HOME by deriving from HOME
    cfgDir = filepath.Join(s.HomeDir, ".config")
    wsFile := filepath.Join(cfgDir, "workspace-manager", "workspaces", name+".json")
    data, err := os.ReadFile(wsFile)
    if err != nil { t.Fatalf("read workspace file: %v", err) }
    var ws struct{ Path string `json:"path"` }
    if err := json.Unmarshal(data, &ws); err != nil { t.Fatalf("parse workspace file: %v", err) }
    return ws.Path
}

// Context returns a cancellable context for subprocess timeouts if needed.
func (s *Sandbox) Context(t *testing.T) context.Context {
    t.Helper()
    return context.Background()
}

func must(err error) {
    if err != nil { panic(err) }
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


