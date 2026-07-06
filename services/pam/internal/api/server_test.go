package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/platform/privilegedsession"
	"era/services/pam/internal/checkout"
	"era/services/pam/internal/kms"
	"era/services/pam/internal/shamir"
	"era/services/pam/internal/vault"
	"era/services/platform/licensegate"
)

func newTestServer(t *testing.T) (*Server, []string) {
	t.Helper()
	v := vault.New(kms.NewSoftwareSealed())
	srv := New(v, checkout.NewStore(), privilegedsession.NewStore(), licensegate.DevAllEnabled(), "software-sealed-dev")
	shares := shamir.EncodeShares(srv.initShares)
	return srv, shares
}

func unseal(t *testing.T, srv *Server, shares []string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"shares": shares[:2]})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vault/unseal", bytes.NewReader(body))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unseal: %d %s", rec.Code, rec.Body.String())
	}
}

func TestVaultSealUnsealAPI(t *testing.T) {
	srv, shares := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vault/status", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	unseal(t, srv, shares)
}

func TestNoSecretLeakInListAPI(t *testing.T) {
	srv, shares := newTestServer(t)
	unseal(t, srv, shares)
	secret := "SuperSecret!Pass-LEAK-TEST"
	body, _ := json.Marshal(map[string]string{
		"name": "test", "password": secret, "username": "root", "tenant_id": "t1",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader(body))
	req.Header.Set("X-ERA-Role", "admin")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("put secret: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
	srv.Routes().ServeHTTP(rec, req)
	if !ResponseMustNotLeak(rec.Body.Bytes(), secret) {
		t.Fatal("secret leaked in list API")
	}
}

func TestCheckoutRevealFlow(t *testing.T) {
	srv, shares := newTestServer(t)
	unseal(t, srv, shares)
	putBody, _ := json.Marshal(map[string]string{
		"name": "ssh", "password": "p@ss", "username": "admin", "target": "srv1",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader(putBody))
	req.Header.Set("X-ERA-Role", "admin")
	srv.Routes().ServeHTTP(rec, req)
	var meta map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &meta)
	secID := meta["id"]

	coBody, _ := json.Marshal(map[string]string{"secret_id": secID})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/checkout", bytes.NewReader(coBody))
	req.Header.Set("X-ERA-Actor", "alice")
	req.Header.Set("X-ERA-Role", "admin")
	srv.Routes().ServeHTTP(rec, req)
	var co map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &co)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/checkout/"+co["id"]+"/reveal", nil)
	req.Header.Set("X-ERA-Actor", "alice")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("reveal: %d %s", rec.Code, rec.Body.String())
	}
}

func TestSSHProxySession(t *testing.T) {
	srv, _ := newTestServer(t)
	body := `{"host":"10.0.0.5"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxy/ssh/start", bytes.NewReader([]byte(body)))
	req.Header.Set("X-ERA-Actor", "ops1")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ssh start: %d", rec.Code)
	}
}

func TestRDPProxySession(t *testing.T) {
	srv, _ := newTestServer(t)
	body := `{"host":"10.0.0.10","port":3389,"width":1920,"height":1080}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxy/rdp/start", bytes.NewReader([]byte(body)))
	req.Header.Set("X-ERA-Actor", "ops1")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("rdp start: %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["protocol"] != "rdp" || out["session_id"] == "" {
		t.Fatalf("%v", out)
	}
}

func TestLDAPApproverGate(t *testing.T) {
	t.Setenv("ERA_LDAP_APPROVER_GROUPS", "pam-approvers")
	srv, shares := newTestServer(t)
	unseal(t, srv, shares)
	putBody, _ := json.Marshal(map[string]string{
		"name": "ssh", "password": "p@ss", "username": "admin", "target": "srv1",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader(putBody))
	req.Header.Set("X-ERA-Role", "analyst")
	srv.Routes().ServeHTTP(rec, req)
	var meta map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &meta)

	coBody, _ := json.Marshal(map[string]string{"secret_id": meta["id"]})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/checkout", bytes.NewReader(coBody))
	req.Header.Set("X-ERA-Actor", "alice")
	req.Header.Set("X-ERA-Role", "analyst")
	srv.Routes().ServeHTTP(rec, req)
	var co map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &co)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/checkout/"+co["id"]+"/approve", nil)
	req.Header.Set("X-ERA-Actor", "bob")
	req.Header.Set("X-ERA-Role", "analyst")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without ldap group, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/checkout/"+co["id"]+"/approve", nil)
	req.Header.Set("X-ERA-Actor", "bob")
	req.Header.Set("X-ERA-Groups", "pam-approvers")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected approve with group, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestCustodyHeadAfterAudit(t *testing.T) {
	srv, shares := newTestServer(t)
	unseal(t, srv, shares)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/custody/head", nil)
	srv.Routes().ServeHTTP(rec, req)
	var out map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	genesis := "0000000000000000000000000000000000000000000000000000000000000000"
	if out["head"] == "" || out["head"] == genesis {
		t.Fatalf("custody head should advance after unseal, got %s", out["head"])
	}
}

func TestPAMLicenseGate(t *testing.T) {
	srv := New(vault.New(kms.NewSoftwareSealed()), checkout.NewStore(), privilegedsession.NewStore(), licensegate.FromModules(nil), "software-sealed-dev")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vault/status", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
