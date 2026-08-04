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
	body := rec.Body.String()
	for _, want := range []string{
		"ERA Webmail",
		`data-line="comms"`,
		`data-sku="mail"`,
		"mail.css",
		"era-chrome.js",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in webmail html", want)
		}
	}
}

func TestMailThemeAssets(t *testing.T) {
	mux := http.NewServeMux()
	NewServer(nil).Register(mux)
	for _, path := range []string{"/mail/static/mail.css", "/mail/static/era-chrome.css", "/mail/static/era-chrome.js"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mail/static/mail.css", nil)
	mux.ServeHTTP(rec, req)
	css := rec.Body.String()
	for _, want := range []string{`tokens/era-theme-comms.css`, `tokens/era-tokens-base.css`} {
		if !strings.Contains(css, want) {
			t.Fatalf("mail.css missing %q", want)
		}
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/mail/static/tokens/era-theme-comms.css", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("comms tokens status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `#188038`) {
		t.Fatal("comms theme missing accent")
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

// Deepen D2: browser OIDC PKCE markers must remain in served app.js (AC-C2 lab).
func TestStaticAppJSContainsPKCE(t *testing.T) {
	mux := http.NewServeMux()
	NewServer(nil).Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mail/static/app.js", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"code_challenge",
		"code_challenge_method",
		"S256",
		"code_verifier",
		"pkce_verifier",
		"/oauth2/authorize",
		"/oauth2/token",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("app.js missing PKCE marker %q", want)
		}
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
