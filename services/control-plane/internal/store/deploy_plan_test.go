package store

import "testing"

func TestPlanPatchesOpenSSL(t *testing.T) {
	st := newMemoryStore()
	st.ReplaceAssetSoftware("n1", "t1", []*AssetSoftware{
		{Name: "OpenSSL Library", Version: "3.0.13"},
	})
	plan := st.PlanPatches(nil)
	if len(plan) == 0 {
		t.Fatal("expected openssl patch")
	}
	if plan[0].CVEID != "CVE-2024-1234" {
		t.Fatalf("cve: %s", plan[0].CVEID)
	}
}
