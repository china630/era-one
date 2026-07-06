package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSQLitePersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cp.db")

	s1, err := NewSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.UpsertAsset(&Asset{NodeID: "n1", TenantID: "t1", Hostname: "host1", Platform: "linux"})
	s1.CreateCase(&Case{ID: "c1", Title: "test case", Status: "new"})
	_ = s1.Close()

	s2, err := NewSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	if len(s2.ListAssets()) != 1 {
		t.Fatal("asset not persisted")
	}
	if len(s2.ListCases()) != 1 {
		t.Fatal("case not persisted")
	}
}

func TestNewFromEnv(t *testing.T) {
	t.Setenv("ERA_STORE_PATH", "")
	if _, err := NewFromEnv(); err != nil {
		t.Fatal(err)
	}
}

func TestNewFromEnvSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env.db")
	t.Setenv("ERA_STORE_PATH", path)
	st, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	st.UpsertAsset(&Asset{NodeID: "x"})
	_ = st.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
