package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/comms/mail/internal/repo"
)

func TestMailSendPolicyDeny413(t *testing.T) {
	m := repo.NewMemory()
	m.PutPolicy("t-demo", repo.InlinePolicy{MaxAttachmentSizeMB: 1})
	srv := NewServer(Config{Repo: m})
	mux := http.NewServeMux()
	srv.registerMailAPI(mux)

	body := `{"from":"a@x.c","to":"b@x.c","subject":"big","body":"` + strings.Repeat("x", 2*1024*1024) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mail/send", strings.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d want 413", rec.Code)
	}
}
