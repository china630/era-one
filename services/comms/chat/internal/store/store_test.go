package store

import (
	"testing"
	"time"
)

func TestStoreJSONFallback(t *testing.T) {
	t.Setenv("ERA_CHAT_DATABASE_URL", "")
	t.Setenv("ERA_COMMS_DATABASE_URL", "")
	dir := t.TempDir()
	t.Setenv("ERA_CHAT_DATA_DIR", dir)
	s := NewFromEnv()
	r := s.CreateRoom("t1", "general")
	if r.ID == "" {
		t.Fatal("empty room id")
	}
	m, ok := s.AddMessage("t1", r.ID, "alice", "hi")
	if !ok || m.Body != "hi" {
		t.Fatalf("msg %v ok=%v", m, ok)
	}
	list := s.ListMessages("t1", r.ID)
	if len(list) != 1 {
		t.Fatalf("list %d", len(list))
	}
	_ = time.Now()
}
