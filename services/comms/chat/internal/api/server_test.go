package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/comms/chat/internal/audit"
	"era/services/comms/chat/internal/store"
	"era/services/comms/internal/httpauth"
)

func chatMux() *http.ServeMux {
	st := store.New()
	aud := audit.NewRecorder()
	s := NewServer(st, aud)
	mux := http.NewServeMux()
	s.Register(mux)
	return mux
}

func withChatHeaders(req *http.Request) {
	req.Header.Set("X-ERA-Tenant", "t-demo")
	req.Header.Set("X-ERA-Role", "chat.user")
}

func TestChatMessageE2E(t *testing.T) {
	t.Setenv("ERA_CHAT_DEV", "1")
	mux := chatMux()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/rooms", bytes.NewReader([]byte(`{"name":"general"}`)))
	withChatHeaders(req)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create room status %d", rec.Code)
	}
	var room struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &room); err != nil || room.ID == "" {
		t.Fatalf("room %+v", rec.Body.String())
	}

	send := map[string]string{"room_id": room.ID, "sender": "alice", "body": "hello"}
	b, _ := json.Marshal(send)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/chat/messages", bytes.NewReader(b))
	withChatHeaders(req)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("send status %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/chat/messages?room_id="+room.ID, nil)
	withChatHeaders(req)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d", rec.Code)
	}
	var msgs []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &msgs); err != nil || len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %s", rec.Body.String())
	}
}

func TestChatRBAC(t *testing.T) {
	t.Setenv("ERA_CHAT_DEV", "")
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "test-secret-32bytes-minimum!!")
	mux := chatMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/messages?room_id=x", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("expected 401/403, got %d", rec.Code)
	}
}

func TestChatJWTTenantBinding(t *testing.T) {
	secret := []byte("test-secret-32bytes-minimum!!")
	t.Setenv("ERA_CHAT_DEV", "")
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", string(secret))
	tok, err := httpauth.MintDevJWT(secret, "tenant-a", "alice", "chat.user")
	if err != nil {
		t.Fatal(err)
	}
	mux := chatMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/rooms", bytes.NewReader([]byte(`{"name":"bound"}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("X-ERA-Tenant", "tenant-b")
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
		t.Fatalf("tenant got %q want tenant-a (header spoof must not win)", room.TenantID)
	}
}
