package repo

import (
	"testing"
)

func TestOpenFromEnvRequiresDSNUnlessMemory(t *testing.T) {
	t.Setenv("ERA_COMMS_DATABASE_URL", "")
	t.Setenv("ERA_MAIL_STORE", "")
	if _, err := OpenFromEnv(); err == nil {
		t.Fatal("expected error without DSN")
	}
	t.Setenv("ERA_MAIL_STORE", "memory")
	b, err := OpenFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.(*Memory); !ok {
		t.Fatalf("want Memory, got %T", b)
	}
}

func TestMemoryDeliverPolicyTooLarge(t *testing.T) {
	m := NewMemory()
	_, _ = m.CreateMailbox("t1", "a@x.c", "pw", 10<<20)
	m.PutPolicy("t1", InlinePolicy{MaxAttachmentSizeMB: 1})
	big := make([]byte, 2*1024*1024)
	_, err := m.DeliverRaw("a@x.c", big, "a@x.c")
	if err == nil || err.Error() != "message too large" {
		t.Fatalf("err=%v", err)
	}
}
