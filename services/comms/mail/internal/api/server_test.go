package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/comms/mail/internal/audit"
	"era/services/comms/mail/internal/coreclient"
	"era/services/comms/mail/internal/policy"
	"era/services/platform/licensegate"
	"era/services/platform/tenant"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	tenants := tenant.NewStore()
	_ = tenants.PutTenant(tenant.Tenant{ID: "t1", Name: "Test", Slug: "test"})
	_ = tenants.PutDomain(tenant.Domain{ID: "d1", TenantID: "t1", FQDN: "mail.gov.az", Primary: true})

	return NewServer(Config{
		Tenants:  tenants,
		Policies: policy.NewStore(),
		Audit:    audit.NewNoop(),
		Core:     coreclient.NewStub(),
		Gate:     licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsMailServer}),
	})
}

func TestHealthz(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	s.Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["service"] != "era-mail-server" {
		t.Fatalf("unexpected: %+v", body)
	}
}

func TestPolicyDefaults(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	s.Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/policy?tenant_id=unknown", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var p policy.InlinePolicy
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.MaxAttachmentSizeMB != 25 {
		t.Fatalf("expected default 25 MB, got %d", p.MaxAttachmentSizeMB)
	}
}

func TestAutodiscover(t *testing.T) {
	s := newTestServer(t)
	mux := http.NewServeMux()
	s.Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/autodiscover/autodiscover.xml?email=alice@mail.gov.az", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/xml; charset=utf-8" {
		t.Fatalf("content-type: %s", ct)
	}
}
