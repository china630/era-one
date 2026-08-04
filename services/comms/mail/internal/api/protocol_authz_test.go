package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/comms/calendar/store"
	"era/services/comms/mail/internal/api"
	"era/services/comms/mail/internal/repo"
	"era/services/platform/licensegate"
)

func protocolMux(t *testing.T) http.Handler {
	t.Helper()
	m := repo.NewMemory()
	_, _ = m.CreateMailbox("t-demo", "alice@mail.gov.az", "pw", 10<<20)
	srv := api.NewServer(api.Config{
		Repo:     m,
		CalStore: store.New(),
		Gate:     licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsMailServer}),
	})
	mux := http.NewServeMux()
	srv.Register(mux)
	return mux
}

func TestProtocolRoutesUnauthorized(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "proto-authz-secret")
	t.Setenv("ERA_PROTOCOL_BASIC_PASSWORD", "")
	t.Setenv("ERA_INTERNAL_TOKEN", "")

	mux := protocolMux(t)
	paths := []string{
		"/caldav/alice@mail.gov.az/",
		"/carddav/alice@mail.gov.az/",
		"/ews/Exchange.asmx",
		"/Microsoft-Server-ActiveSync",
	}
	for _, p := range paths {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, p, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: want 401, got %d body=%s", p, rec.Code, rec.Body.String())
		}
	}
}

func TestProtocolRoutesDevBypass(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "1")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "")
	t.Setenv("ERA_PROTOCOL_BASIC_PASSWORD", "")

	mux := protocolMux(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/caldav/alice@mail.gov.az/", nil))
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("DEV should not 401, got %d", rec.Code)
	}
}

func TestProtocolRoutesBasicPassword(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "proto-authz-secret")
	t.Setenv("ERA_PROTOCOL_BASIC_PASSWORD", "lab-as-pass")
	t.Setenv("ERA_INTERNAL_TOKEN", "")

	mux := protocolMux(t)
	req := httptest.NewRequest(http.MethodOptions, "/Microsoft-Server-ActiveSync", nil)
	req.SetBasicAuth("alice@mail.gov.az", "lab-as-pass")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("Basic+password should not 401, got %d %s", rec.Code, rec.Body.String())
	}

	ews := httptest.NewRequest(http.MethodPost, "/ews/Exchange.asmx", nil)
	ews.SetBasicAuth("alice@mail.gov.az", "lab-as-pass")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, ews)
	if rec2.Code == http.StatusUnauthorized {
		t.Fatalf("EWS Basic should not 401, got %d", rec2.Code)
	}
}
