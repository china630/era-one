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

func TestCaseLifecycle(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())

	// create
	body := `{"title":"Test incident","detection_id":"d1","node_id":"n1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d", rec.Code)
	}
	var created map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id, _ := created["id"].(string)

	// assign + close
	patch := `{"status":"assigned","assignee":"analyst1"}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/cases/"+id, bytes.NewReader([]byte(patch)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/cases/"+id, bytes.NewReader([]byte(`{"status":"closed"}`)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("close: %d", rec.Code)
	}
}

func TestAssetRegister(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())
	body := `{"agent_id":"a1","tenant_id":"t1","node_id":"n1","hostname":"h1","platform":"windows"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assets/register", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("register: %d", rec.Code)
	}
	if len(st.ListAssets()) != 1 {
		t.Fatal("expected 1 asset")
	}
}
