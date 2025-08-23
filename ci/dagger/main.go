package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"dagger.io/dagger"
)

func main() {
	var (
		backendsStr = flag.String("backends", "hybrid", "comma-separated backends: hybrid,cli,gogit")
		race       = flag.Bool("race", false, "enable -race")
		coverage   = flag.Bool("cover", false, "enable coverage")
		smoke      = flag.Bool("smoke", false, "run a smaller subset with -run")
		outDir     = flag.String("out", ".out", "host artifacts directory")
	)
	flag.Parse()

	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(log.Writer()))
	if err != nil { log.Fatal(err) }
	defer client.Close()

	backends := strings.Split(*backendsStr, ",")

	src := client.Host().Directory(".")

	base := client.Container().From("golang:1.24.4").
		WithMountedCache("/go/pkg/mod", client.CacheVolume("gomod")).
		WithMountedCache("/root/.cache/go-build", client.CacheVolume("gobuild")).
		WithWorkdir("/workspace").
		WithMountedDirectory("/workspace", src)

	// Install git and update certs
	base = base.WithExec([]string{"bash", "-lc", "apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*"})

	// Ensure output dir inside container and setup HOME and git identity
	base = base.WithEnvVariable("HOME", "/tmp/wsm-home")
	base = base.WithExec([]string{"bash", "-lc", "mkdir -p .out /tmp/wsm-home/.config"})

	goBin := "/usr/local/go/bin/go"

	for _, be := range backends {
		fmt.Printf("\n=== Running backend: %s ===\n", be)
		c := base.WithEnvVariable("WSM_GIT_BACKEND", be)

		// Build wsm
		c = c.WithExec([]string{"bash", "-lc", goBin + " build -o .out/wsm ./cmd/wsm"})

		// Build test command (guard if tests folder missing)
		baseTest := goBin + " test ./test/integration/... -v -count=1"
		if *race { baseTest += " -race" }
		if *coverage { baseTest += " -coverprofile=.out/coverage-" + be + ".out" }
		if *smoke { baseTest += " -run 'Test(Smoke|Status|Diff)'" }
		fullCmd := "if [ -d test/integration ]; then " + baseTest + "; else echo 'No integration tests found, skipping.'; fi > .out/test-" + be + ".log 2>&1"

		// Run tests (or skip) and write logs
		c = c.WithExec([]string{"bash", "-lc", fullCmd})

		// Export artifacts
		if _, err := c.Directory("/workspace/.out").Export(ctx, *outDir); err != nil {
			log.Fatalf("export artifacts: %v", err)
		}
	}

	fmt.Println("Dagger pipeline completed.")
	if _, err := os.Stat(*outDir); err == nil {
		fmt.Printf("Artifacts exported to %s\n", *outDir)
	}
}
