package repo

import "testing"

func TestMemoryMailboxAndDeliver(t *testing.T) {
	m := NewMemory()
	mb, err := m.CreateMailbox("t1", "alice@example.com", "secret", 1<<20)
	if err != nil || mb.Email != "alice@example.com" {
		t.Fatalf("create: %v", err)
	}
	if !m.VerifyMailboxPassword("alice@example.com", "secret") {
		t.Fatal("password verify failed")
	}
	if m.VerifyMailboxPassword("alice@example.com", "wrong") {
		t.Fatal("wrong password accepted")
	}
	msg, err := m.DeliverRaw("alice@example.com", []byte("hello"), "bob@example.com")
	if err != nil || msg.UID != 1 {
		t.Fatalf("deliver: %v uid=%d", err, msg.UID)
	}
	msgs, err := m.ListMessages("alice@example.com")
	if err != nil || len(msgs) != 1 {
		t.Fatalf("list: %v len=%d", err, len(msgs))
	}
}

func TestMemoryCalendar(t *testing.T) {
	m := NewMemory()
	body := "BEGIN:VCALENDAR\r\nEND:VCALENDAR"
	ev, err := m.PutCalendarEvent("alice@example.com", "evt-1", body)
	if err != nil || ev.UID != "evt-1" {
		t.Fatalf("put: %v", err)
	}
	got, ok := m.GetCalendarEvent("alice@example.com", "evt-1")
	if !ok || got.Body != body {
		t.Fatal("get failed")
	}
}

func TestMemoryEWSMessage(t *testing.T) {
	m := NewMemory()
	_, _ = m.CreateMailbox("t1", "bob@example.com", "pw", 1<<20)
	msg, err := m.AddEWSMessage("bob@example.com", "Hi", "Body")
	if err != nil || msg.Subject != "Hi" {
		t.Fatalf("ews: %v", err)
	}
	got, ok := m.GetMessageByID("bob@example.com", formatMsgID(msg.ID))
	if !ok {
		t.Fatal("get by id")
	}
	if got.Body != "Body" {
		t.Fatalf("body=%q", got.Body)
	}
}
