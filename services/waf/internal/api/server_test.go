package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/platform/licensegate"
	"era/services/waf/internal/rules"
)

func TestProxyAndBlock(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("upstream-ok:" + r.URL.Path))
	}))
	defer up.Close()

	srv, err := New(rules.NewOWASP(), up.URL, licensegate.DevAllEnabled(), nil, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "upstream-ok") {
		t.Fatalf("proxy: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/search?q=%27+OR+select+", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("block: %d %s", rec.Code, rec.Body.String())
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "era-waf-sqli") {
		t.Fatalf("body %s", body)
	}
}

func TestLicenseGate(t *testing.T) {
	srv, err := New(rules.NewOWASP(), "http://127.0.0.1:9", licensegate.FromModules(nil), nil, 1024)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/waf/rules", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d", rec.Code)
	}
}

func TestRuleCRUD(t *testing.T) {
	srv, err := New(rules.NewOWASP(), "http://127.0.0.1:9", licensegate.DevAllEnabled(), nil, 1024)
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Routes()
	before := len(srv.Engine.Rules())

	body := `{"id":"era-waf-custom","category":"custom","severity":"medium","pattern":"(?i)evil-token"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/waf/rules", strings.NewReader(body)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("post: %d %s", rec.Code, rec.Body.String())
	}
	if len(srv.Engine.Rules()) != before+1 {
		t.Fatalf("rules count after add: %d", len(srv.Engine.Rules()))
	}

	upd := `{"category":"custom","severity":"high","pattern":"(?i)evil-token-2"}`
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/v1/waf/rules/era-waf-custom", strings.NewReader(upd)))
	if rec.Code != http.StatusOK {
		t.Fatalf("put: %d %s", rec.Code, rec.Body.String())
	}
	got, ok := srv.Engine.GetRule("era-waf-custom")
	if !ok || got.Severity != "high" {
		t.Fatalf("updated rule: %+v", got)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/waf/rules/era-waf-custom", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	if _, ok := srv.Engine.GetRule("era-waf-custom"); ok {
		t.Fatal("rule still present after delete")
	}
}
