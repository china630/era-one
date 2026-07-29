package chat

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatUIShell(t *testing.T) {
	mux := http.NewServeMux()
	NewServer().Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/chat", nil)
	req.Header.Set("X-ERA-Tenant", "t-demo")
	req.Header.Set("X-ERA-Role", "chat.user")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
