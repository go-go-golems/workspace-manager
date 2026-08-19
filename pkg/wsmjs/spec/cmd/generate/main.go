package main

import (
	"bytes"
	"fmt"
	"os"
)

func main() {
	templatePath := "wsm.d.ts.tmpl"
	outputPath := "wsm.d.ts"

	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		panic(fmt.Errorf("read template %s: %w", templatePath, err))
	}

	current, err := os.ReadFile(outputPath)
	if err == nil && bytes.Equal(current, templateData) {
		return
	}

	if err := os.WriteFile(outputPath, templateData, 0o644); err != nil { // #nosec G703 -- generated spec output written to a caller-supplied path in the repo; no secret content.
		panic(fmt.Errorf("write output %s: %w", outputPath, err))
	}
}
