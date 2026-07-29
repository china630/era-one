package mail

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthz(t *testing.T) {
	mux := http.NewServeMux()
	NewServer(nil).Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mail/healthz", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMailIndexServesHTML(t *testing.T) {
	mux := http.NewServeMux()
	NewServer(nil).Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mail", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ERA Webmail") {
		t.Fatalf("expected webmail html")
	}
}

func TestStaticAppJS(t *testing.T) {
	mux := http.NewServeMux()
	NewServer(nil).Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mail/static/app.js", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestWithJWTRejectsMissingToken(t *testing.T) {
	mux := http.NewServeMux()
	NewServer(nil).Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mail/api/messages", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestWithJWTRejectsInvalidToken(t *testing.T) {
	mux := http.NewServeMux()
	NewServer(nil).Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mail/api/messages", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestWithJWTAcceptsValidToken(t *testing.T) {
	secret := []byte("dev-only-change-in-prod")
	tok, err := SignTestToken(secret, "t-demo", "alice@mail.gov.az", "u-alice", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewServer(nil).Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mail/api/messages", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Fatal("valid token rejected")
	}
}
