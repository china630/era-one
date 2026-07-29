package audit

import (
	"context"
	"testing"

	erav1 "era/contracts/gen/era/v1"
)

func TestNoopInsert(t *testing.T) {
	w := NewNoop()
	if err := w.Insert(context.Background(), &erav1.MailAuditEvent{
		TenantId: "t1",
		Mailbox:  "alice@mail.gov.az",
		Action:   erav1.MailAuditAction_MAIL_AUDIT_SEND,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRecordSendNoop(t *testing.T) {
	w := NewNoop()
	if err := w.RecordSend(context.Background(), "t1", "alice@mail.gov.az", "<msg@test>", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
}
