package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/comms/chat/internal/audit"
	"era/services/comms/chat/internal/store"
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
	mux := chatMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/messages?room_id=x", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
}
