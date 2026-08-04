package oidc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"era/services/platform/internal/oidc"
)

func TestStagingTokenDevOnly(t *testing.T) {
	t.Setenv("ERA_IDENTITY_DEV", "1")
	s, err := oidc.NewServer("http://test", []byte("secret"), "")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.Register(mux)

	body := `{"email":"alice@mail.gov.az","password":"1234"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth2/staging/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.AccessToken == "" {
		t.Fatal("missing access_token")
	}
}

func TestStagingTokenDisabledWithoutDev(t *testing.T) {
	os.Unsetenv("ERA_IDENTITY_DEV")
	s, err := oidc.NewServer("http://test", []byte("secret"), "")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth2/staging/token", strings.NewReader(`{}`))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 without dev, got %d", rec.Code)
	}
}

func TestStagingRegisterThenToken(t *testing.T) {
	t.Setenv("ERA_IDENTITY_DEV", "1")
	s, err := oidc.NewServer("http://test", []byte("secret"), "")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.Register(mux)

	email := "new.user@lab.local"
	body := `{"email":"` + email + `","password":"secret1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth2/staging/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register status %d body=%s", rec.Code, rec.Body.String())
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/oauth2/staging/register", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate, got %d", rec2.Code)
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/oauth2/staging/token", strings.NewReader(body))
	req3.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("token after register status %d", rec3.Code)
	}
}
