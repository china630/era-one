package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadyzNoopClickHouse(t *testing.T) {
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
