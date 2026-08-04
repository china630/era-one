package mitre

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"era/services/detection-engine/internal/sigma"
)

func TestCorpusCoverageGolden(t *testing.T) {
	rules := []*sigma.Rule{
		{ID: "r1", Tags: []string{"attack.T1099"}},
		{ID: "r2", Tags: []string{"attack.T1099", "attack.T1059.001"}},
		{ID: "r3", Tags: []string{"os.windows"}},
	}
	got := CorpusCoverage(rules)
	data, _ := json.MarshalIndent(got, "", "  ")
	golden := filepath.Join("testdata", "coverage.golden.json")
	want, err := os.ReadFile(golden)
	if err != nil {
		_ = os.MkdirAll("testdata", 0o755)
		_ = os.WriteFile(golden, data, 0o644)
		t.Logf("wrote %s", golden)
		return
	}
	var a, b any
	_ = json.Unmarshal(data, &a)
	_ = json.Unmarshal(want, &b)
	ag, _ := json.Marshal(a)
	bg, _ := json.Marshal(b)
	if string(ag) != string(bg) {
		t.Fatalf("mismatch\ngot  %s\nwant %s", ag, bg)
	}
}
