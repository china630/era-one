package hybrid

import (
	"testing"
	"time"

	"era/services/control-plane/internal/store"
)

func TestBuildHealthANoForbiddenFields(t *testing.T) {
	st := store.NewMemory()
	st.UpsertAsset(&store.Asset{NodeID: "n1", TenantID: "t1", Hostname: "h1", Platform: "linux"})
	st.CreateCase(&store.Case{ID: "c1", Title: "test", Status: "new"})
	pol := store.HybridPolicy{DeploymentID: "dep-1", TenantID: "t1"}
	rt := store.HybridRuntime{LeaseStatus: "VALID"}
	h := BuildHealthA(st, pol, rt)
	if h.AgentCount != 1 || h.CaseCount != 1 {
		t.Fatalf("counts: %+v", h)
	}
	raw := `{"level":"A","deployment_id":"dep-1"}`
	if !ContainsForbiddenEgress(`{"cmdline":"evil"}`) {
		t.Fatal("should detect forbidden")
	}
	if ContainsForbiddenEgress(raw) {
		t.Fatal("should allow health A")
	}
	red := RedactHealthJSON(`contact admin@bank.az for help`)
	if red == `contact admin@bank.az for help` {
		t.Fatal("expected email redaction")
	}
}

func TestHostAllowed(t *testing.T) {
	list := []string{"portal.era.local", "update.era.local"}
	if !HostAllowed(list, "portal.era.local") {
		t.Fatal("expected allowed")
	}
	if HostAllowed(list, "evil.example.com") {
		t.Fatal("expected blocked")
	}
}

func TestDefaultHybridPolicyAirGap(t *testing.T) {
	p := store.DefaultHybridPolicy()
	if p.Enabled {
		t.Fatal("connected must be OFF by default")
	}
	if p.HealthLevel != "A" {
		t.Fatalf("health level: %s", p.HealthLevel)
	}
	_ = time.Now()
}
