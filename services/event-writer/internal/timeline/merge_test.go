package timeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeGoldenMultiSource(t *testing.T) {
	events := []Entry{
		{Kind: "event", At: "2026-07-01T10:00:00Z", NodeID: "n1", Severity: "medium", Category: "process", Source: "agent", Summary: "cmd.exe"},
		{Kind: "event", At: "2026-07-01T10:00:05Z", NodeID: "n1", Severity: "low", Category: "network", Source: "agent", Summary: "outbound 443"},
		{Kind: "event", At: "2026-07-01T10:00:08Z", NodeID: "n1", Severity: "medium", Category: "auth", Source: "byo-edr", Summary: "failed logon"},
	}
	detections := []Entry{
		{Kind: "detection", At: "2026-07-01T10:00:03Z", NodeID: "n1", Severity: "high", Source: "sigma", Summary: "Suspicious PowerShell", DetectionID: "det-1"},
	}
	got := Merge(events, detections)
	b, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "timeline_merged.golden.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(golden, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Skipf("golden missing: %v (run UPDATE_GOLDEN=1)", err)
	}
	if string(b)+"\n" != string(want) && string(b) != string(want) {
		t.Fatalf("timeline golden mismatch; run UPDATE_GOLDEN=1\n got:\n%s", string(b))
	}
	if len(got) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(got))
	}
	if got[0].Category != "process" || got[len(got)-1].Category != "auth" {
		t.Fatalf("order wrong: %+v", got)
	}
}
