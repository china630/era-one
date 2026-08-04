package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/comms/internal/httpauth"
	"era/services/comms/mail/internal/audit"
	"era/services/comms/mail/internal/repo"
)

func TestMailSendRecordsAudit(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "1")
	m := repo.NewMemory()
	_, _ = m.CreateMailbox("t-demo", "a@x.c", "pw", 1<<20)
	_, _ = m.CreateMailbox("t-demo", "b@x.c", "pw", 1<<20)
	aud := audit.NewNoop()
	srv := NewServer(Config{Repo: m, Audit: aud})
	mux := http.NewServeMux()
	srv.registerMailAPI(mux, httpauth.FromEnv("ERA_MAIL_DEV", ""))

	body := `{"from":"a@x.c","to":"b@x.c","subject":"hi","body":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mail/send", strings.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMailSendPolicyDeny413(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "1")
	m := repo.NewMemory()
	_, _ = m.CreateMailbox("t-demo", "a@x.c", "pw", 1<<20)
	m.PutPolicy("t-demo", repo.InlinePolicy{MaxAttachmentSizeMB: 1})
	srv := NewServer(Config{Repo: m})
	mux := http.NewServeMux()
	srv.registerMailAPI(mux, httpauth.FromEnv("ERA_MAIL_DEV", ""))

	body := `{"from":"a@x.c","to":"b@x.c","subject":"big","body":"` + strings.Repeat("x", 2*1024*1024) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mail/send", strings.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d want 413", rec.Code)
	}
}

func TestMailSendUnauthorizedWithoutDev(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "test-secret-32bytes-minimum!!")
	m := repo.NewMemory()
	srv := NewServer(Config{Repo: m})
	mux := http.NewServeMux()
	srv.registerMailAPI(mux, httpauth.FromEnv("ERA_MAIL_DEV", ""))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mail/send", strings.NewReader(`{}`))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}
