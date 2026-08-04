package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/comms/internal/httpauth"
	"era/services/comms/vcs/internal/adapter"
	"era/services/comms/vcs/internal/audit"
	"era/services/comms/vcs/internal/store"
)

func vcsMux() *http.ServeMux {
	st := store.New()
	aud := audit.NewRecorder()
	s := NewServer(st, adapter.Stub{}, aud)
	mux := http.NewServeMux()
	s.Register(mux)
	return mux
}

func withVCSHeaders(req *http.Request) {
	req.Header.Set("X-ERA-Tenant", "t-demo")
	req.Header.Set("X-ERA-Role", "vcs.user")
}

func TestConferenceRoomJoin(t *testing.T) {
	t.Setenv("ERA_VCS_DEV", "1")
	mux := vcsMux()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/rooms", bytes.NewReader([]byte(`{"name":"standup"}`)))
	withVCSHeaders(req)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create room %d %s", rec.Code, rec.Body.String())
	}
	var room struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &room); err != nil || room.ID == "" {
		t.Fatal(err)
	}

	tokenBody := map[string]string{"room_id": room.ID, "participant": "alice"}
	b, _ := json.Marshal(tokenBody)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/vcs/token", bytes.NewReader(b))
	withVCSHeaders(req)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("token %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token":"lk-token-`) {
		t.Fatalf("unexpected token body %s", rec.Body.String())
	}
}

func TestVCSUnauthNoToken(t *testing.T) {
	t.Setenv("ERA_VCS_DEV", "")
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "test-secret-32bytes-minimum!!")
	t.Setenv("ERA_INTERNAL_TOKEN", "")
	mux := vcsMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/rooms", bytes.NewReader([]byte(`{"name":"x"}`)))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("expected 401/403, got %d", rec.Code)
	}
}

func TestVCSSpoofHeadersRejectedWithoutDev(t *testing.T) {
	t.Setenv("ERA_VCS_DEV", "")
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "test-secret-32bytes-minimum!!")
	mux := vcsMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/rooms", bytes.NewReader([]byte(`{"name":"evil"}`)))
	req.Header.Set("X-ERA-Tenant", "evil-tenant")
	req.Header.Set("X-ERA-Role", "vcs.user")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("spoof want 401/403, got %d", rec.Code)
	}
}

func TestVCSJWTTenantBinding(t *testing.T) {
	secret := []byte("test-secret-32bytes-minimum!!")
	t.Setenv("ERA_VCS_DEV", "")
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", string(secret))
	tok, err := httpauth.MintDevJWT(secret, "tenant-a", "alice", "vcs.user")
	if err != nil {
		t.Fatal(err)
	}
	mux := vcsMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/rooms", bytes.NewReader([]byte(`{"name":"bound"}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("X-ERA-Tenant", "tenant-b") // spoof must not win
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	var room struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	if room.TenantID != "tenant-a" {
		t.Fatalf("tenant got %q want tenant-a", room.TenantID)
	}
}
