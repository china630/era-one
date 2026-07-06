package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/platform/licensegate"
	"era/services/service-desk/internal/store"
)

func TestIncidentLifecycle(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)

	body := `{"title":"VPN down","node_id":"n1","requester":"user1","sla_hours":4}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var inc map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &inc)
	id, _ := inc["id"].(string)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/incidents/"+id, bytes.NewReader([]byte(`{"status":"in_progress","assignee":"tech1"}`)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d", rec.Code)
	}
}

func TestServiceRequestPortal(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	body := `{"title":"New laptop","requester":"user2","category":"hardware"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/requests", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("request: %d", rec.Code)
	}
}

func TestLicenseGateService(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.FromModules(nil), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
