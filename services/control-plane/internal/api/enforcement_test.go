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

func TestEnforcementWriteRequiresAdmin(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "dev")
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())
	body := `{"version":"1.0.1","mode":"enforce","fail_mode":"open","app_rules":[],"device_rules":[],"virtual_patches":[]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/enforcement/policy", bytes.NewReader([]byte(body)))
	req.Header.Set("X-ERA-Role", "viewer")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer put want 403 got %d", rec.Code)
	}
}

func TestEnforcementSpoofAdminRejectedInProxyTrust(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "proxy")
	t.Setenv("ERA_API_KEY", "")
	t.Setenv("ERA_TRUSTED_PROXY_CIDRS", "")
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())
	body := `{"version":"1.0.1","mode":"enforce","fail_mode":"open","app_rules":[],"device_rules":[],"virtual_patches":[]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/enforcement/policy", bytes.NewReader([]byte(body)))
	req.Header.Set("X-ERA-Role", "admin") // spoof without Trusted-Proxy
	req.RemoteAddr = "203.0.113.10:443"
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("spoof admin want 401 got %d %s", rec.Code, rec.Body.String())
	}
	// Trusted-Proxy header from untrusted hop must still fail.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/enforcement/policy", bytes.NewReader([]byte(body)))
	req.Header.Set("X-ERA-Role", "admin")
	req.Header.Set("X-ERA-Trusted-Proxy", "1")
	req.RemoteAddr = "203.0.113.10:443"
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("header-only Trusted-Proxy want 401 got %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/enforcement/policy", bytes.NewReader([]byte(body)))
	req.Header.Set("X-ERA-Role", "admin")
	req.Header.Set("X-ERA-Trusted-Proxy", "1")
	req.RemoteAddr = "127.0.0.1:12345"
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("trusted put: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAgentPolicyGetSpoofRejectedInProxyTrust(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "proxy")
	t.Setenv("ERA_TRUSTED_PROXY_CIDRS", "")
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/enforcement/policy", nil)
	req.Header.Set("X-ERA-Actor", "era-agent")
	req.RemoteAddr = "203.0.113.10:443"
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("spoof agent want 401 got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/enforcement/policy", nil)
	req.Header.Set("X-ERA-Actor", "era-agent")
	req.Header.Set("X-ERA-Trusted-Proxy", "1")
	req.RemoteAddr = "127.0.0.1:9"
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("trusted agent: %d", rec.Code)
	}
}

func TestAgentPolicyGetWithAPIKey(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "api_key")
	t.Setenv("ERA_API_KEY", "agent-secret")
	t.Setenv("ERA_AGENT_TOKEN", "")
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/enforcement/policy", nil)
	req.Header.Set("X-ERA-Actor", "era-agent")
	req.Header.Set("X-ERA-Role", "admin") // forge ignored
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("role forge want 401 got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/enforcement/policy", nil)
	req.Header.Set("X-ERA-Actor", "era-agent")
	req.Header.Set("Authorization", "Bearer agent-secret")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("bearer API key: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAgentTokenPolicyGetAllowedWritesForbidden(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "api_key")
	t.Setenv("ERA_API_KEY", "admin-secret")
	t.Setenv("ERA_AGENT_TOKEN", "agent-secret")
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/enforcement/policy", nil)
	req.Header.Set("Authorization", "Bearer agent-secret")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("agent GET policy want 200 got %d %s", rec.Code, rec.Body.String())
	}

	body := `{"version":"1.0.1","mode":"enforce","fail_mode":"open","app_rules":[],"device_rules":[],"virtual_patches":[]}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/enforcement/policy", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer agent-secret")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("agent PUT policy want 403 got %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/enforcement/escrow/n1/C:", nil)
	req.Header.Set("Authorization", "Bearer agent-secret")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("agent escrow detail want 403 got %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/enforcement/policy", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer admin-secret")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("API key PUT policy want 200 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestAuditSpoofAdminRejectedInProxyTrust(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "proxy")
	t.Setenv("ERA_TRUSTED_PROXY_CIDRS", "")
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	req.Header.Set("X-ERA-Role", "admin")
	req.RemoteAddr = "203.0.113.10:443"
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("spoof audit want 401 got %d", rec.Code)
	}
}
