package sigma

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchContains(t *testing.T) {
	r := &Rule{
		ID:    "test-1",
		Title: "Test",
		Logsource: map[string]string{"category": "process"},
		Detection: map[string]any{
			"selection": map[string]any{"CommandLine|contains": "powershell -enc"},
			"condition": "selection",
		},
	}
	if !r.Match("process", `{"command_line":"powershell -enc ABC"}`) {
		t.Fatal("expected match")
	}
	if r.Match("network", `powershell -enc`) {
		t.Fatal("wrong category should not match when logsource set")
	}
}

func TestLoadDir(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "..", "data", "sigma-corpus", "rules")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("corpus not generated yet")
	}
	rules, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) < 10 {
		t.Fatalf("expected rules, got %d", len(rules))
	}
	if errs := Lint(rules); len(errs) > 0 {
		t.Fatalf("lint: %v", errs[:min(3, len(errs))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
