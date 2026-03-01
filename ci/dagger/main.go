package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"dagger.io/dagger"
)

func main() {
	var (
		race     = flag.Bool("race", false, "enable -race")
		coverage = flag.Bool("cover", false, "enable coverage")
		smoke    = flag.Bool("smoke", false, "run a smaller subset with -run")
		outDir   = flag.String("out", ".out", "host artifacts directory")
	)
	flag.Parse()

	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(log.Writer()))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	src := client.Host().Directory(".", dagger.HostDirectoryOpts{
		Include: []string{"**/*"},
		Exclude: []string{
			".git/**",
			"**/.git/**",
			"**/.cache/**",
			"**/.out/**",
		},
	})

	base := client.Container().From("golang:1.24.4").
		WithMountedCache("/go/pkg/mod", client.CacheVolume("gomod")).
		WithMountedCache("/root/.cache/go-build", client.CacheVolume("gobuild")).
		WithWorkdir("/workspace").
		WithMountedDirectory("/workspace", src)

	// Install git and update certs
	base = base.WithExec([]string{"bash", "-lc", "apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*"})

	// Ensure output dir inside container and setup HOME
	base = base.WithEnvVariable("HOME", "/tmp/wsm-home")
	base = base.WithExec([]string{"bash", "-lc", "mkdir -p .out /tmp/wsm-home/.config"})

	// Diagnostics: capture directory and test visibility
	base = base.WithExec([]string{"bash", "-lc", `pwd; ls -la; echo '--- dirs depth=2 ---'; find . -maxdepth 2 -type d -print | sort; echo '--- tests ---'; (find test -maxdepth 4 -type f -name '*_test.go' -print || true) | sort | tee .out/diag.txt`})

	goBin := "/usr/local/go/bin/go"

	backend := "cli"
	fmt.Printf("\n=== Running backend: %s ===\n", backend)
	c := base

	// Build wsm
	c = c.WithExec([]string{"bash", "-lc", goBin + " build -o .out/wsm ./cmd/wsm"})

	// Build test command (no guard: fail loudly and capture output)
	baseTest := goBin + " test ./test/integration/... -v -count=1"
	if *race {
		baseTest += " -race"
	}
	if *coverage {
		baseTest += " -coverprofile=.out/coverage-" + backend + ".out"
	}
	if *smoke {
		baseTest += " -run 'Test(Smoke|Status|Diff)'"
	}
	fullCmd := "set -euo pipefail; " + baseTest + " | tee .out/test-" + backend + ".log"

	// Run tests and write logs
	c = c.WithExec([]string{"bash", "-lc", fullCmd})

	// Export artifacts
	if _, err := c.Directory("/workspace/.out").Export(ctx, *outDir); err != nil {
		log.Fatalf("export artifacts: %v", err)
	}

	fmt.Println("Dagger pipeline completed.")
	if _, err := os.Stat(*outDir); err == nil {
		fmt.Printf("Artifacts exported to %s\n", *outDir)
	}
}
