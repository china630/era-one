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

func TestTechniquesFromTags(t *testing.T) {
	r := &Rule{Tags: []string{"attack.t1099", "attack.T1059.001", "os.windows"}}
	got := r.Techniques()
	want := []string{"T1099", "T1059.001"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestMitreEmitGolden(t *testing.T) {
	r := &Rule{
		ID:    "era-timetest-timestomp",
		Title: "Timestomp",
		Level: "medium",
		Tags:  []string{"attack.T1099"},
		Logsource: map[string]string{"category": "process"},
		Detection: map[string]any{
			"selection": map[string]any{"CommandLine|contains": "timestomp"},
			"condition": "selection",
		},
	}
	tech := r.Techniques()
	if len(tech) != 1 || tech[0] != "T1099" {
		t.Fatalf("%v", tech)
	}
	if !r.Match("process", `timestomp.exe -r`) {
		t.Fatal("expected match")
	}
}
