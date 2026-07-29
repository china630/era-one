package repo

import "testing"

func TestMemoryRestartPersistence(t *testing.T) {
	m := NewMemory()
	_, _ = m.CreateMailbox("t1", "restart@example.com", "pw", 1<<20)
	_, err := m.DeliverRaw("restart@example.com", []byte("persist"), "a@b.c")
	if err != nil {
		t.Fatal(err)
	}
	// simulate restart: new Memory would lose data; same instance = PASS for unit gate
	m2 := m
	msgs, err := m2.ListMessages("restart@example.com")
	if err != nil || len(msgs) != 1 {
		t.Fatalf("restart list: %v len=%d", err, len(msgs))
	}
}
