package internalapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/comms/mail/internal/internalapi"
	"era/services/comms/mail/internal/repo"
)

func TestPolicyByMailbox(t *testing.T) {
	m := repo.NewMemory()
	_, _ = m.CreateMailbox("t1", "bob@x.c", "pw", 100<<20)
	m.PutPolicy("t1", repo.InlinePolicy{MaxAttachmentSizeMB: 2, QuotaMBPerUser: 100})
	h := &internalapi.Handler{Repo: m}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/mail/policy?email=bob@x.c", nil)
	h.PolicyHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if int(out["max_message_bytes"].(float64)) != 2*1024*1024 {
		t.Fatalf("max_message_bytes=%v", out["max_message_bytes"])
	}
}
