package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
