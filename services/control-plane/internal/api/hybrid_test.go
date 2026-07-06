package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/control-plane/internal/rbac"
	"era/services/control-plane/internal/store"
	"era/services/platform/licensegate"
)

func TestHybridPolicyAirGapDefault(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevDefault())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hybrid/policy", nil)
	req.Header.Set("X-ERA-Role", "admin")
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var resp struct {
		Policy store.HybridPolicy `json:"policy"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Policy.Enabled {
		t.Fatal("expected connected OFF")
	}
}

func TestHybridPolicyPutRequiresAdmin(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevDefault())
	body, _ := json.Marshal(map[string]any{
		"enabled":      true,
		"portal_url":   "https://portal.era.local",
		"deployment_id": "dep-1",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hybrid/policy", bytes.NewReader(body))
	req.Header.Set("X-ERA-Role", string(rbac.RoleViewer))
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestHybridPolicyPutAdmin(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevDefault())
	body, _ := json.Marshal(map[string]any{
		"enabled":       true,
		"portal_url":    "https://portal.era.local",
		"update_url":    "https://update.era.local",
		"deployment_id": "dep-1",
		"license_id":    "lic-1",
		"egress_allowlist": []string{"portal.era.local", "update.era.local"},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/hybrid/policy", bytes.NewReader(body))
	req.Header.Set("X-ERA-Role", string(rbac.RoleAdmin))
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	if !st.GetHybridPolicy().Enabled {
		t.Fatal("expected enabled")
	}
	audit := st.ListAudit(10)
	if len(audit) == 0 {
		t.Fatal("expected audit entry")
	}
}

func TestHybridStatus(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevDefault())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/hybrid/status", nil)
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
}
