package hybrid

import (
	"encoding/json"
	"testing"
	"time"

	"era/services/control-plane/internal/store"
)

func TestBuildHealthBNoPII(t *testing.T) {
	st := store.NewMemory()
	st.UpsertAsset(&store.Asset{
		NodeID: "n1", TenantID: "t1", Hostname: "secret-host.internal",
		Platform: "linux", LastSeen: time.Now().UTC(),
	})
	st.CreateCase(&store.Case{ID: "c1", Title: "case", Status: "new"})
	pol := store.HybridPolicy{DeploymentID: "dep-b", TenantID: "t1", HealthLevel: "B"}
	rt := store.HybridRuntime{LeaseStatus: "VALID", LastBundleID: "bnd-1"}
	h := BuildHealthB(st, pol, rt)
	if h.Level != "B" || h.AgentCount != 1 || h.OpenCases != 1 {
		t.Fatalf("health B: %+v", h)
	}
	body, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	safe := RedactHealthJSON(string(body))
	if ContainsForbiddenEgress(safe) || HealthBForbiddenFields(safe) {
		t.Fatalf("health B failed PII gate: %s", safe)
	}
	if HealthBForbiddenFields(`{"password":"x"}`) == false {
		t.Fatal("should block password field")
	}
}

func TestApplyBundleBumpsPolicy(t *testing.T) {
	st := store.NewMemory()
	before := st.Policy().Version
	applyBundleToPolicy(st, &bundleClaims{BundleID: "bnd-test", Kind: "cve-feed"})
	after := st.Policy()
	if after.Version == before {
		t.Fatalf("version not bumped: %s", after.Version)
	}
	if after.Rules["cve-feed"] != "bundle:bnd-test" {
		t.Fatalf("rules: %+v", after.Rules)
	}
}

func TestBumpPolicyVersion(t *testing.T) {
	got := bumpPolicyVersion("3.0.0-ga", "bnd-99")
	if got != "3.0.1+bnd-99" {
		t.Fatalf("got %s", got)
	}
}
