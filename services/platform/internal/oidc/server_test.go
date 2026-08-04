package oidc_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/platform/internal/oidc"
)

func TestDiscovery(t *testing.T) {
	s, err := oidc.NewServer("http://test", []byte("secret"), "")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !contains(rec.Body.String(), "authorization_endpoint") {
		t.Fatal("missing discovery fields")
	}
}

func TestPKCEPair(t *testing.T) {
	v, c := oidc.NewPKCEVerifier()
	if v == "" || c == "" || v == c {
		t.Fatal("pkce pair invalid")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || index(s, sub) >= 0)
}

func index(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
