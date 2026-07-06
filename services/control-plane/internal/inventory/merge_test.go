package inventory

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"era/services/control-plane/internal/store"
)

func TestMergeGoldenScenarios(t *testing.T) {
	st := store.NewMemory()
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	st.UpsertAssetFull(&store.Asset{
		NodeID: "node-a", TenantID: "t1", Hostname: "ws01", AgentID: "agent-1",
		SerialNumber: "SN-100", MACAddrs: []string{"aa:bb:cc:dd:ee:01"},
		LastSeen: now,
	})

	cases := []struct {
		name     string
		snap     Snapshot
		wantID   string
		wantRule string
	}{
		{
			name:     "agent_id_match",
			snap:     Snapshot{TenantID: "t1", AgentID: "agent-1", Hostname: "ws01-renamed", NodeID: "node-new"},
			wantID:   "node-a",
			wantRule: "agent_id",
		},
		{
			name:     "serial_after_reinstall",
			snap:     Snapshot{TenantID: "t1", AgentID: "agent-2", SerialNumber: "SN-100", NodeID: "node-b"},
			wantID:   "node-a",
			wantRule: "serial",
		},
		{
			name:     "mac_overlap",
			snap:     Snapshot{TenantID: "t1", MACAddrs: []string{"aa:bb:cc:dd:ee:01"}, Hostname: "unknown", NodeID: "node-c"},
			wantID:   "node-a",
			wantRule: "mac",
		},
		{
			name:     "hostname_hint",
			snap:     Snapshot{TenantID: "t1", Hostname: "ws01", NodeID: "node-d"},
			wantID:   "node-a",
			wantRule: "hostname",
		},
	}

	type row struct {
		Case   string `json:"case"`
		NodeID string `json:"node_id"`
		Rule   string `json:"rule"`
	}
	var got []row
	for _, tc := range cases {
		id, rule := ResolveNodeID(st, tc.snap)
		if id != tc.wantID || rule != tc.wantRule {
			t.Fatalf("%s: got id=%s rule=%s want %s %s", tc.name, id, rule, tc.wantID, tc.wantRule)
		}
		got = append(got, row{Case: tc.name, NodeID: id, Rule: rule})
	}

	golden := filepath.Join("testdata", "merge_scenarios.golden.json")
	b, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		_ = os.MkdirAll("testdata", 0o755)
		_ = os.WriteFile(golden, append(b, '\n'), 0o644)
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Skipf("golden missing: %v", err)
	}
	var gotCanon, wantCanon any
	_ = json.Unmarshal(b, &gotCanon)
	_ = json.Unmarshal(bytes.TrimSpace(want), &wantCanon)
	if !jsonEqual(gotCanon, wantCanon) {
		t.Fatalf("golden mismatch; UPDATE_GOLDEN=1\n%s", string(b))
	}
}

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}
