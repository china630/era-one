package api

import (
	"encoding/json"
	"net/http"
	"os"

	"era/services/comms/chat/internal/audit"
	"era/services/comms/chat/internal/store"
	"era/services/comms/internal/httpauth"
)

type Server struct {
	Store   *store.Store
	Auditor *audit.Recorder
}

func NewServer(st *store.Store, aud *audit.Recorder) *Server {
	return &Server{Store: st, Auditor: aud}
}

func (s *Server) Register(mux *http.ServeMux) {
	devKey := "ERA_CHAT_DEV"
	if os.Getenv("ERA_CHAT_DEV") != "1" && os.Getenv("ERA_MAIL_DEV") == "1" {
		devKey = "ERA_MAIL_DEV"
	}
	auth := httpauth.FromEnv(devKey, "chat.user")
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/v1/chat/rooms", auth.Wrap(s.createRoom))
	mux.HandleFunc("/api/v1/chat/messages", auth.Wrap(s.messages))
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	mode := "memory"
	if s.Store != nil {
		mode = s.Store.Backend()
	}
	writeJSON(w, map[string]string{"status": "ok", "service": "era-chat", "storage_mode": mode})
}

// tenantID uses principal from httpauth.Wrap (JWT claims / DEV / internal).
// Never prefers spoofable X-ERA-Tenant over JWT when authenticated via JWT.
func tenantID(r *http.Request) string {
	if p, ok := httpauth.FromContext(r.Context()); ok && p.TenantID != "" {
		return p.TenantID
	}
	return ""
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
	tenant := tenantID(r)
	if tenant == "" {
		http.Error(w, "tenant required", http.StatusUnauthorized)
		return
	}
	room := s.Store.CreateRoom(tenant, body.Name)
	writeJSON(w, room)
}

func (s *Server) messages(w http.ResponseWriter, r *http.Request) {
	tenant := tenantID(r)
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
		if s.Auditor != nil {
			s.Auditor.Record(audit.Event{Action: "CHAT_MESSAGE", TenantID: tenant, RoomID: body.RoomID})
		}
		writeJSON(w, m)
	case http.MethodGet:
		roomID := r.URL.Query().Get("room_id")
		writeJSON(w, s.Store.ListMessages(tenant, roomID))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
