package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestDocsAIWithoutJWT401(t *testing.T) {
	s := &server{jwtSecret: []byte("s"), licenseOK: true}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/docs-ai/summarize", strings.NewReader("hello"))
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestDocsAIWithoutLicense403(t *testing.T) {
	s := &server{jwtSecret: []byte("s"), licenseOK: false}
	tok := mustJWT(t, "s", "t1", "u1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/docs-ai/summarize", strings.NewReader("hello"))
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
}

func TestDocsAISummarizeJSONBody(t *testing.T) {
	s := &server{jwtSecret: []byte("test-secret"), licenseOK: true}
	tok := mustJWT(t, "test-secret", "t1", "u1")
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/docs-ai/summarize",
		strings.NewReader(`{"text":"json body notes"}`),
	)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "json body notes") || !strings.Contains(rr.Body.String(), `"mode":"stub"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestDocsAIRewriteJSONBody(t *testing.T) {
	s := &server{jwtSecret: []byte("test-secret"), licenseOK: true}
	tok := mustJWT(t, "test-secret", "t1", "u1")
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/docs-ai/rewrite",
		strings.NewReader(`{"text":"draft paragraph","instruction":"formal tone"}`),
	)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "draft paragraph") || !strings.Contains(body, `"mode":"stub"`) {
		t.Fatalf("body=%s", body)
	}
	if !strings.Contains(body, "stub rewrite") {
		t.Fatalf("want stub rewrite prefix, body=%s", body)
	}
}

func TestDocsAIStubModeNoPhoneHome(t *testing.T) {
	dialed := false
	s := &server{
		jwtSecret: []byte("s"),
		licenseOK: true,
		ollamaURL: "", // unset → stub; must not dial
		dialed:    &dialed,
		client: &http.Client{Transport: roundTripFn(func(*http.Request) (*http.Response, error) {
			dialed = true
			t.Fatal("stub path must not dial")
			return nil, nil
		})},
	}
	tok := mustJWT(t, "s", "t1", "u1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/docs-ai/summarize", strings.NewReader("agenda notes"))
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["mode"] != "stub" {
		t.Fatalf("want mode=stub, got %+v", got)
	}
	if dialed {
		t.Fatal("stub path dialed network")
	}
	out := s.summarize("x")
	if out["mode"] != "stub" || dialed {
		t.Fatalf("summarize stub leaked dial: %+v dialed=%v", out, dialed)
	}
}

func TestDocsAIAllowlistHostOnly(t *testing.T) {
	c, err := allowlistedClient("http://127.0.0.1:11434", nil)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, "http://evil.example/api/generate", nil)
	_, err = c.Do(req)
	if err == nil {
		t.Fatal("expected allowlist block")
	}
	if !strings.Contains(err.Error(), "air-gap") {
		t.Fatalf("want air-gap block, got %v", err)
	}
}

func TestLicenseFailClosedInProduction(t *testing.T) {
	t.Setenv("ERA_PRODUCTION", "1")
	t.Setenv("ERA_OFFICE_DEV", "")
	t.Setenv("ERA_LICENSE_OFFICE_AI", "")
	if licenseFromEnv() {
		t.Fatal("expected fail-closed")
	}
}

func mustJWT(t *testing.T, secret, tenant, sub string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenant_id": tenant,
		"sub":       sub,
	})
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

type roundTripFn func(*http.Request) (*http.Response, error)

func (f roundTripFn) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
