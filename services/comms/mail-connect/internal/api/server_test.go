package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/comms/mail-connect/internal/audit"
	syncstore "era/services/comms/mail-connect/internal/sync"
)

func newMux() (*http.ServeMux, *audit.Recorder) {
	store := syncstore.NewStore()
	aud := audit.NewRecorder()
	s := NewServer(store, aud)
	mux := http.NewServeMux()
	s.Register(mux)
	return mux, aud
}

func TestRegisterAndSync(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "1")
	mux, aud := newMux()

	body, _ := json.Marshal(map[string]string{
		"tenant_id":    "t-demo",
		"email":        "alice@mail.gov.az",
		"provider":     "imap",
		"username":     "alice",
		"password_ref": "vault://alice",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connect/mailboxes", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}

	startBody := `{"tenant_id":"t-demo","mailbox":"alice@mail.gov.az"}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/connect/sync", strings.NewReader(startBody))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var j syncstore.Job
	if err := json.Unmarshal(rec.Body.Bytes(), &j); err != nil {
		t.Fatal(err)
	}
	if j.ID == "" || j.Status != "done" {
		t.Fatalf("unexpected job %+v", j)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/connect/sync?id="+j.ID, nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	if aud.Count() < 2 {
		t.Fatalf("expected audit events >=2, got %d", aud.Count())
	}
}

func TestRegisterAndSyncUnauthorized(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_CONNECT_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "test-secret-32bytes-minimum!!")
	mux, _ := newMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connect/mailboxes", strings.NewReader(`{}`))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestConnectAutodiscover(t *testing.T) {
	mux, _ := newMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/connect/autodiscover.xml?email=alice@mail.gov.az", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "<Type>CONNECT</Type>") {
		t.Fatalf("missing CONNECT protocol: %s", rec.Body.String())
	}
}

