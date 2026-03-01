package registry

import (
	"reflect"
	"testing"

	"github.com/go-go-golems/glazed/pkg/types"
)

func TestDiscoverResultToRow(t *testing.T) {
	result := &discoverExecutionResult{
		Paths:           []string{"/tmp/a", "/tmp/b"},
		RepositoryCount: 3,
	}

	row := discoverResultToRow(result)
	m := types.RowToMap(row)

	paths, ok := m["paths"].([]string)
	if !ok {
		t.Fatalf("paths field has unexpected type: %T", m["paths"])
	}
	if !reflect.DeepEqual(paths, result.Paths) {
		t.Fatalf("unexpected paths: got %v want %v", paths, result.Paths)
	}

	repoCount, ok := m["repository_count"].(int)
	if !ok {
		t.Fatalf("repository_count field has unexpected type: %T", m["repository_count"])
	}
	if repoCount != result.RepositoryCount {
		t.Fatalf("unexpected repository_count: got %d want %d", repoCount, result.RepositoryCount)
	}
}
