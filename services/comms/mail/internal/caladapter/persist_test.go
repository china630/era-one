package caladapter

import (
	"testing"

	"era/services/comms/mail/internal/repo"
)

func TestCalendarSurvivesMemoryRepoRoundTrip(t *testing.T) {
	m := repo.NewMemory()
	_, _ = m.CreateMailbox("t-demo", "alice@mail.gov.az", "pw", 10<<20)
	cal := New(m)
	ev := cal.Put("alice@mail.gov.az", "uid-1", "BEGIN:VCALENDAR\r\nEND:VCALENDAR")
	if ev == nil {
		t.Fatal("put returned nil")
	}
	got, ok := cal.Get("alice@mail.gov.az", "uid-1")
	if !ok || got.UID != ev.UID {
		t.Fatalf("calendar miss after put: ok=%v got=%v", ok, got)
	}
	list := cal.List("alice@mail.gov.az")
	if len(list) != 1 {
		t.Fatalf("list=%d", len(list))
	}
}
