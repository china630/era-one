package api

import (
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/cloud-portal/internal/store"
	lic "era/services/license/pkg/license"
)

func TestLeaseRenewAndCRL(t *testing.T) {
	pub, priv, err := lic.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}
	st := store.New()
	st.UpsertInstallation(&store.Installation{
		DeploymentID: "dep-1", LicenseID: "lic-1", TenantID: "t1",
	})
	srv := New(st, priv, pub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hybrid/lease/renew?deployment_id=dep-1&license_id=lic-1", nil)
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("lease status %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/hybrid/crl", nil)
	w2 := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("crl status %d", w2.Code)
	}
}

func TestHealthRejectsRaw(t *testing.T) {
	pub, priv, err := lic.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}
	srv := New(store.New(), priv, pub)
	body := `{"deployment_id":"d1","raw_event":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hybrid/health", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHealthAcceptsLevelA(t *testing.T) {
	pub, priv, _ := lic.GenerateKeypair()
	srv := New(store.New(), priv, pub)
	body := `{"deployment_id":"d1","level":"A","agent_count":5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hybrid/health", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	_ = ed25519.PublicKey(pub)
}

func TestManagedView(t *testing.T) {
	pub, priv, _ := lic.GenerateKeypair()
	st := store.New()
	st.UpsertInstallation(&store.Installation{DeploymentID: "d1"})
	srv := New(st, priv, pub)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/managed/summary", nil)
	req.Header.Set("X-ERA-Role", "partner")
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
}
