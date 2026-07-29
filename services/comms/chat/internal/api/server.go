package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"era/services/comms/chat/internal/audit"
	"era/services/comms/chat/internal/store"
)

type Server struct {
	Store   *store.Store
	Auditor *audit.Recorder
}

func NewServer(st *store.Store, aud *audit.Recorder) *Server {
	return &Server{Store: st, Auditor: aud}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/v1/chat/rooms", s.withRBAC(s.createRoom))
	mux.HandleFunc("/api/v1/chat/messages", s.messages)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "service": "era-chat"})
}

func (s *Server) withRBAC(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authorize(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func authorize(r *http.Request) bool {
	tenant := r.Header.Get("X-ERA-Tenant")
	role := r.Header.Get("X-ERA-Role")
	return tenant != "" && strings.Contains(role, "chat.user")
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	tenant := r.Header.Get("X-ERA-Tenant")
	room := s.Store.CreateRoom(tenant, body.Name)
	writeJSON(w, room)
}

func (s *Server) messages(w http.ResponseWriter, r *http.Request) {
	if !authorize(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tenant := r.Header.Get("X-ERA-Tenant")
	switch r.Method {
	case http.MethodPost:
		var body struct {
			RoomID string `json:"room_id"`
			Sender string `json:"sender"`
			Body   string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RoomID == "" || body.Sender == "" {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		m, ok := s.Store.AddMessage(tenant, body.RoomID, body.Sender, body.Body)
		if !ok {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}
		s.Auditor.Record(audit.Event{Action: "CHAT_MESSAGE", TenantID: tenant, RoomID: body.RoomID, User: body.Sender})
		writeJSON(w, m)
	case http.MethodGet:
		roomID := r.URL.Query().Get("room_id")
		if roomID == "" {
			http.Error(w, "room_id required", http.StatusBadRequest)
			return
		}
		writeJSON(w, s.Store.ListMessages(tenant, roomID))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
