package store

import (
	"os"
	"path/filepath"
	"testing"
)

func runParityScenario(t *testing.T, st Repository) {
	t.Helper()
	st.UpsertAsset(&Asset{NodeID: "n1", TenantID: "t1", Hostname: "h1", Platform: "linux"})
	st.UpsertAsset(&Asset{NodeID: "n2", TenantID: "t2", Hostname: "h2", Platform: "windows"})
	st.CreateCase(&Case{ID: "c1", Title: "case-a", Status: "new", TenantID: "t1"})
	st.CreateCase(&Case{ID: "c2", Title: "case-b", Status: "open", TenantID: "t2"})

	if len(st.ListAssets()) != 2 {
		t.Fatalf("assets: got %d", len(st.ListAssets()))
	}
	if len(st.ListCases()) != 2 {
		t.Fatalf("cases: got %d", len(st.ListCases()))
	}
	if st.Policy().Version == "" {
		t.Fatal("policy version empty")
	}
	pol := st.GetHybridPolicy()
	if pol.HealthLevel != "A" {
		t.Fatalf("hybrid health level: %s", pol.HealthLevel)
	}
	st.SetHybridPolicy(HybridPolicy{Enabled: true, PortalURL: "https://p.local"})
	if !st.GetHybridPolicy().Enabled {
		t.Fatal("hybrid policy set failed")
	}
	st.RecordEgressAudit(&EgressAuditEntry{Kind: "test", Target: "p.local", Level: "A", Bytes: 1})
	if len(st.ListEgressAudit(5)) == 0 {
		t.Fatal("egress audit empty")
	}
}

func TestSQLitePostgresParitySQLite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "parity.db")
	st, err := NewSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	runParityScenario(t, st)
}

func TestPostgresParity(t *testing.T) {
	dsn := os.Getenv("ERA_STORE_PG_DSN")
	if dsn == "" {
		t.Skip("ERA_STORE_PG_DSN not set")
	}
	st, err := NewPostgres(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	runParityScenario(t, st)
}

func TestTenantScopeFiltersAssets(t *testing.T) {
	t.Setenv("ERA_MULTI_TENANT", "1")
	st := newMemoryStore()
	st.UpsertAsset(&Asset{NodeID: "a1", TenantID: "tenant-a"})
	st.UpsertAsset(&Asset{NodeID: "a2", TenantID: "tenant-b"})
	scoped := Scoped(st, "tenant-a").(*memoryStore)
	if len(scoped.ListAssets()) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(scoped.ListAssets()))
	}
}
