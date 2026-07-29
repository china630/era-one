package audit_test

import (
	"testing"

	"era/services/comms/mail-moderation/internal/audit"
)

func TestMemory_Record(t *testing.T) {
	m := &audit.Memory{}
	if err := m.Record(audit.Event{HoldID: "h1", Action: "hold", Sender: "a@b.c"}); err != nil {
		t.Fatal(err)
	}
	if len(m.Events) != 1 || m.Events[0].Action != "hold" {
		t.Fatalf("%+v", m.Events)
	}
}
