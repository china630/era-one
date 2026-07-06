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

func TestEnforcementPolicyRoundtrip(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/enforcement/policy", nil)
	req.Header.Set("X-ERA-Actor", "era-agent")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get policy: %d %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	pol, _ := got["policy"].(map[string]any)
	if pol["mode"] != "monitor" {
		t.Fatalf("expected monitor, got %v", pol["mode"])
	}

	body := `{"version":"1.0.1","mode":"enforce","fail_mode":"open","app_rules":[],"device_rules":[],"virtual_patches":[]}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/enforcement/policy", bytes.NewReader([]byte(body)))
	req.Header.Set("X-ERA-Role", "admin")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put policy: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/enforcement/rollback", nil)
	req.Header.Set("X-ERA-Role", "admin")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rollback: %d", rec.Code)
	}
}

func TestVirtualPatchAPI(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())

	body := `{"cve_id":"CVE-2024-9999","hash_sha256":"abcd1234ef567890abcd1234ef567890abcd1234ef567890abcd1234ef56","path":"*\\evil.dll","vector":"dll_load","mode":"monitor"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enforcement/virtual-patch", bytes.NewReader([]byte(body)))
	req.Header.Set("X-ERA-Role", "admin")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("virtual patch: %d %s", rec.Code, rec.Body.String())
	}
	pol := st.GetEnforcementPolicy()
	if len(pol.VirtualPatches) < 2 {
		t.Fatalf("expected virtual_patches appended, got %d", len(pol.VirtualPatches))
	}
	if len(pol.AppRules) < 2 {
		t.Fatal("expected hash app rule")
	}
}

func TestBitlockerEscrowMaskedList(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())
	st.UpsertBitlockerEscrow(&store.BitlockerEscrow{
		NodeID: "n1", TenantID: "t1", VolumeID: "C:", KeyBlob: "secret-key",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/enforcement/escrow?node_id=n1", nil)
	req.Header.Set("X-ERA-Role", "admin")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list escrow: %d", rec.Code)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("secret-key")) {
		t.Fatal("key leaked in list response")
	}
}
