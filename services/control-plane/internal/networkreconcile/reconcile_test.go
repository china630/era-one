package networkreconcile

import (
	"testing"

	"era/services/control-plane/internal/store"
)

func TestUpsertNetworkDedupIP(t *testing.T) {
	st := store.NewMemory()
	in := Input{
		NodeID: "net-10-0-0-1", TenantID: "t1", Hostname: "switch-core",
		Kind: "switch", IPAddrs: []string{"10.0.0.1"}, MACAddrs: []string{"00:11:22:33:44:01"},
	}
	r1 := Upsert(st, in)
	if r1.Conflict || r1.Asset == nil {
		t.Fatalf("%+v", r1)
	}
	in2 := Input{NodeID: "net-10-0-0-1", TenantID: "t1", IPAddrs: []string{"10.0.0.1"}, Kind: "switch"}
	r2 := Upsert(st, in2)
	if len(ListNetwork(st)) != 1 {
		t.Fatalf("expected dedup, got %d", len(ListNetwork(st)))
	}
	if r2.Asset.NodeID != r1.Asset.NodeID {
		t.Fatalf("dedup failed")
	}
}

func TestUpsertNetworkManagedFalse(t *testing.T) {
	st := store.NewMemory()
	r := Upsert(st, Input{
		NodeID: "net-10-0-0-9", TenantID: "t1", Kind: "switch",
		IPAddrs: []string{"10.0.0.9"},
	})
	if r.Asset == nil || r.Asset.Managed {
		t.Fatalf("network asset should be unmanaged, got %+v", r.Asset)
	}
	if r.Asset.AssetKind != "switch" {
		t.Fatalf("asset_kind=%q", r.Asset.AssetKind)
	}
}

func TestManagedIPConflict(t *testing.T) {
	st := store.NewMemory()
	st.UpsertAssetFull(&store.Asset{
		NodeID: "ep-1", TenantID: "t1", AgentID: "agent-1",
		Hostname: "ws-1", Platform: "windows", IPAddrs: []string{"10.0.0.50"},
	})
	r := Upsert(st, Input{
		NodeID: "net-10-0-0-50", TenantID: "t1", IPAddrs: []string{"10.0.0.50"},
	})
	if !r.Conflict {
		t.Fatal("expected conflict with managed endpoint")
	}
}
