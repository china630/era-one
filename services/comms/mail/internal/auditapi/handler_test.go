package auditapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/comms/mail/internal/audit"
	erav1 "era/contracts/gen/era/v1"
)

func TestPostAuditSend(t *testing.T) {
	h := &Handler{Writer: audit.NewNoop()}
	body := `{"tenant_id":"t1","mailbox":"bob@mail.gov.az","action":"send","mail_from":"alice@mail.gov.az"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/audit", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestPostAuditBadMethod(t *testing.T) {
	h := &Handler{Writer: audit.NewNoop()}
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/audit", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestPostAuditInsertNoop(t *testing.T) {
	w := audit.NewNoop()
	if err := w.Insert(context.Background(), &erav1.MailAuditEvent{
		TenantId: "t1",
		Mailbox:  "a@b.c",
		Action:   erav1.MailAuditAction_MAIL_AUDIT_SEND,
	}); err != nil {
		t.Fatal(err)
	}
}
