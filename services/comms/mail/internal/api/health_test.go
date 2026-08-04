package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadyzNoopClickHouse(t *testing.T) {
	t.Setenv("ERA_MAIL_AUDIT_REQUIRE", "0")
	s := newTestServer(t)
	mux := http.NewServeMux()
	s.Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Ready  bool              `json:"ready"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.Ready {
		t.Fatalf("expected ready: %+v", body)
	}
	if body.Checks["clickhouse"] != "disabled" {
		t.Fatalf("clickhouse check: %q", body.Checks["clickhouse"])
	}
}

func TestReadyzAuditRequiredWithoutCH(t *testing.T) {
	t.Setenv("ERA_MAIL_AUDIT_REQUIRE", "1")
	s := newTestServer(t) // Audit noop
	mux := http.NewServeMux()
	s.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d want 503 body=%s", rec.Code, rec.Body.String())
	}
}
