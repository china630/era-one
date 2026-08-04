package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/control-plane/internal/store"
	"era/services/platform/licensegate"
)

func TestSuppressionsCRUD(t *testing.T) {
	srv := New(store.NewMemory(), licensegate.DevAllEnabled())
	body, _ := json.Marshal(map[string]string{"rule_id": "era-fp", "node_id": "n1", "reason": "benign"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/suppressions", bytes.NewReader(body))
	req.Header.Set("X-ERA-Actor", "analyst")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal(created)
	}
	rec = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/suppressions", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	rec = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/suppressions/"+id, nil))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
}
