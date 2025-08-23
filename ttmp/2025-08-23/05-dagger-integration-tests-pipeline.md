### 1. Purpose

This document designs a full Dagger-based pipeline to run `workspace-manager` (WSM) integration tests in a reproducible containerized environment, locally and in CI, without hand-rolling Docker scripts.

---

### 2. Why Dagger for WSM

- Single, portable pipeline definition in Go.
- Reproducible containerized environment (pin images, no host leakage).
- Built-in caching (Go mod + build cache) to keep iteration fast.
- Easy matrices (backends: `cli`, `gogit`, `hybrid`), variants (race, coverage, smoke vs full).
- Artifact export (binaries, logs, JUnit) to host or CI.

---

### 3. High-level Goals

- Build `wsm` and run integration tests inside a controlled container.
- Support backend matrix (`WSM_GIT_BACKEND`), with default `hybrid` and optional `cli`/`gogit` passes.
- Provide flags for: race detector, coverage, smoke vs full suite, and parallel jobs.
- Export logs, junit (optional), and binary artifacts as needed.
- Run the same pipeline locally (`go run ./ci/dagger`) and in CI.

---

### 4. Repo Layout (proposed additions)

- `ci/dagger/main.go` → entrypoint to run the pipeline.
- `ci/dagger/pipeline.go` → pipeline construction (optionally inline with main.go if small).
- `test/integration/` → integration tests (helpers/scenarios, see 04-integration-tests-design.md).
- `Makefile` → targets: `dagger`, `dagger-test`, `dagger-test-backends`.

---

### 5. Dagger Setup

- Dependency: `go get dagger.io/dagger@v0.11.3` (or latest stable used by your environment).
- Runtime: Dagger engine via Docker (installed on host/CI runner).
- Base image (pin digest where possible): `golang:1.24-bullseye` + `git`.

---

### 6. Container Environment

- Base container: `golang:1.24-bullseye`.
- Install: `git`, `bash`, `ca-certificates` (and optional mergetools for UI-free tests).
- Workdir: `/workspace`.
- Caches:
  - `/go/pkg/mod` → Go modules cache.
  - `/root/.cache/go-build` → Go build cache.
- Env:
  - `HOME=/tmp/wsm-home` (isolated), write minimal `~/.gitconfig` with test user/email.
  - `WSM_GIT_BACKEND` set per matrix run.

---

### 7. Pipeline Design (Go, Dagger SDK)

- Steps (per backend variant):
  1) Prepare base container + install deps + mount code + mount caches.
  2) `go mod download` (cached).
  3) Build `wsm` (optionally export artifact).
  4) Run integration tests `go test ./test/integration/...` with flags:
     - `-v -count=1`
     - optional `-race` (toggle)
     - optional `-run` for smoke subset.
  5) Export logs, optional coverage/junit (if using gotestsum or `-json` parsing).

- Matrices:
  - Iterate over `[]string{"hybrid","cli","gogit"}`; subset via CLI flags.

- Artifacts:
  - Test logs to `/workspace/.out/logs/<backend>.txt`.
  - Optional: `coverage.out`, `junit.xml` if using gotestsum.

---

### 8. Example Pipeline (Go)

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"dagger.io/dagger"
)

func main() {
	var (
		backendsStr = flag.String("backends", "hybrid", "comma-separated backends: hybrid,cli,gogit")
		race       = flag.Bool("race", false, "enable -race")
		coverage   = flag.Bool("cover", false, "enable coverage")
		smoke      = flag.Bool("smoke", false, "run a smaller subset with -run")
	)
	flag.Parse()

	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(log.Writer()))
	if err != nil { log.Fatal(err) }
	defer client.Close()

	backends := strings.Split(*backendsStr, ",")

	// Host project directory
	src := client.Host().Directory(".")

	// Base container with deps
	base := client.Container().From("golang:1.24-bullseye").
		WithExec([]string{"bash", "-lc", "apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*"}).
		WithMountedCache("/go/pkg/mod", client.CacheVolume("gomod")).
		WithMountedCache("/root/.cache/go-build", client.CacheVolume("gobuild")).
		WithWorkdir("/workspace").
		WithMountedDirectory("/workspace", src)

	// Write minimal git config
	base = base.WithExec([]string{"bash", "-lc", "mkdir -p /tmp/wsm-home && printf '[user]\n\tname = test\n\temail = test@example.com\n' > /tmp/wsm-home/.gitconfig"}).
	base = base.WithEnvVariable("HOME", "/tmp/wsm-home")

	// Download modules once (shared across matrix)
	base = base.WithExec([]string{"bash", "-lc", "go mod download"})

	for _, be := range backends {
		fmt.Printf("\n=== Running backend: %s ===\n", be)
		c := base.WithEnvVariable("WSM_GIT_BACKEND", be)

		// Build wsm binary (optional export)
		build := c.WithExec([]string{"bash", "-lc", "go build -o .out/wsm ./cmd/wsm"})

		// Construct go test command
		testCmd := "go test ./test/integration/... -v -count=1"
		if *race { testCmd += " -race" }
		if *coverage { testCmd += " -coverprofile=.out/coverage-" + be + ".out" }
		if *smoke { testCmd += " -run Test(Smoke|Status|Diff)" } // adjust filter

		// Run tests
		run := build.WithExec([]string{"bash", "-lc", testCmd + " | tee .out/test-" + be + ".log"})

		// Export artifacts to host (optional; remove if CI collects via logs)
		_, err := run.Directory("/workspace/.out").Export(ctx, ".out")
		if err != nil { log.Fatalf("export artifacts: %v", err) }
	}
}
```

Notes:
- Dagger caches module and build data across runs.
- Artifacts land under host `./.out/` by default.
- Adjust `-run` filters and test package path once helpers/scenarios exist.

---

### 9. Local Usage

- Build and run:
  - `go run ./ci/dagger --backends=hybrid,cli,gogit --race`
  - `go run ./ci/dagger --backends=hybrid --smoke`
- Makefile targets:
```
dagger:
	go run ./ci/dagger --backends=hybrid

dagger-test:
	go run ./ci/dagger --backends=hybrid,cli,gogit --race
```

---

### 10. CI Integration (GitHub Actions example)

```yaml
name: integration-tests
on:
  push:
    branches: [ main ]
  pull_request:

jobs:
  dagger:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24.x'
      - name: Set up Docker (for Dagger engine)
        uses: docker/setup-buildx-action@v3
      - name: Run Dagger pipeline (hybrid smoke)
        run: go run ./ci/dagger --backends=hybrid --smoke
      - name: Upload artifacts
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: wsm-integration-out
          path: ./.out
```

For nightly or a separate job, run full backends with `--race` and coverage.

---

### 11. Enhancements

- JUnit XML: use `gotestsum --junitfile .out/junit-<be>.xml --` to produce CI-friendly reports.
- Coverage merge: merge per-backend coverage files; upload to coverage service.
- Pin base image by digest for reproducibility.
- Add a `smoke` package with very fast end-to-end checks for PR gating; nightly executes the full suite.
- Parameterize Go version via env/flag.

---

### 12. Security & Secrets

- No external secrets required for current tests.
- If future tests need tokens (e.g., GitHub), inject via Dagger secrets API rather than env files.

---

### 13. Risks & Mitigations

- Engine availability: Ensure Docker or compatible Dagger engine present on CI runner.
- Long runs: Use caching and smoke/full split; shard matrix if needed.
- Flakiness: keep hermetic repos and avoid network-dependent steps.

---

### 14. Implementation TODOs

- [ ] Add Dagger dependency and `ci/dagger` program.
- [ ] Implement base container (install git, caches, HOME setup).
- [ ] Add backend matrix, build, and test steps; export `.out/` artifacts.
- [ ] Wire Makefile targets and a GitHub Actions workflow.
- [ ] Integrate with `test/integration` helpers and scenarios from 04-design.
- [ ] Optional: gotestsum for JUnit + coverage aggregation.
- [ ] Pin base image digest; consider OCI cache for faster CI.
