package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/platform/cpclient"
	"era/services/platform/licensegate"
	"era/services/provision/internal/store"
)

func TestEnrollRegistersAsset(t *testing.T) {
	var registered bool
	cpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/assets/register" {
			registered = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"policy_version":"1"}`))
		}
	}))
	defer cpSrv.Close()

	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), cpclient.New(cpSrv.URL))
	body := `{"agent_id":"a1","node_id":"n-prov-1","hostname":"bare-metal-01","platform":"linux"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("enroll: %d %s", rec.Code, rec.Body.String())
	}
	if !registered {
		t.Fatal("expected CMDB register")
	}
}

func TestPXEConfigGolden(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pxe/config", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pxe: %d", rec.Code)
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["default_image"] != "img-linux-22" {
		t.Fatalf("default_image: %v", out["default_image"])
	}
}

func TestImagesList(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("images: %d", rec.Code)
	}
}
