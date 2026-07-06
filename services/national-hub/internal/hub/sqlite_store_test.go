package hub

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSQLiteStorePersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taxii.db")
	s1, err := OpenSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.Publish(DefaultCollection, "org-a", "obj-1", []byte(`{"type":"indicator"}`))
	s1.Subscribe("org-b", DefaultCollection)
	_ = s1.Close()

	s2, err := OpenSQLiteStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if s2.PublishCount(DefaultCollection) != 1 {
		t.Fatal("object not persisted")
	}
	if s2.SubscriberCount(DefaultCollection) != 1 {
		t.Fatal("subscriber not persisted")
	}
	objs := s2.Poll(DefaultCollection)
	if len(objs) != 1 || objs[0].ID != "obj-1" {
		t.Fatalf("poll: %+v", objs)
	}
}

func TestNewStoreFromEnv(t *testing.T) {
	st, cleanup, err := NewFromEnv("")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if st.PublishCount(DefaultCollection) != 0 {
		t.Fatal("expected empty memory store")
	}
}

func TestNewStoreFromEnvSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env.db")
	t.Setenv("ERA_STORE_PATH", path)
	st, cleanup, err := NewFromEnv(path)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	st.Publish(DefaultCollection, "o", "1", []byte(`{}`))
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
