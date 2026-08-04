package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"era/services/comms/internal/httpauth"
	"era/services/comms/vcs/internal/adapter"
	"era/services/comms/vcs/internal/audit"
	"era/services/comms/vcs/internal/store"
)

type Server struct {
	Store   *store.Store
	Adapter adapter.LiveKitAdapter
	Auditor *audit.Recorder
}

func NewServer(st *store.Store, lk adapter.LiveKitAdapter, aud *audit.Recorder) *Server {
	return &Server{Store: st, Adapter: lk, Auditor: aud}
}

func (s *Server) Register(mux *http.ServeMux) {
	// Lab: ERA_VCS_DEV or ERA_MAIL_DEV; prod: JWT/internal required (fail-closed).
	devKey := "ERA_VCS_DEV"
	if os.Getenv("ERA_VCS_DEV") != "1" && os.Getenv("ERA_MAIL_DEV") == "1" {
		devKey = "ERA_MAIL_DEV"
	}
	auth := httpauth.FromEnv(devKey, "vcs.user")
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/v1/vcs/rooms", auth.Wrap(s.createRoom))
	mux.HandleFunc("/api/v1/vcs/token", auth.Wrap(s.issueToken))
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	mode := "stub"
	if os.Getenv("ERA_LIVEKIT_URL") != "" {
		mode = "livekit"
	}
	writeJSON(w, map[string]string{"status": "ok", "service": "era-conference", "mode": mode})
}

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
	lkRoom, err := s.Adapter.CreateRoom(body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	room := store.Room{
		ID:        s.Store.NextID(),
		TenantID:  tenant,
		Name:      body.Name,
		LKRoom:    lkRoom,
		CreatedAt: time.Now().UTC(),
	}
	s.Store.Put(room)
	writeJSON(w, room)
}

func (s *Server) issueToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		RoomID      string `json:"room_id"`
		Participant string `json:"participant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RoomID == "" || body.Participant == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	tenant := tenantID(r)
	room, ok := s.Store.Get(body.RoomID)
	if !ok || room.TenantID != tenant {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}
	token, err := s.Adapter.IssueToken(room.LKRoom, body.Participant)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.Auditor.Record(audit.Event{Action: "VCS_ROOM_JOIN", TenantID: tenant, RoomID: body.RoomID, User: body.Participant})
	writeJSON(w, map[string]string{"token": token, "room": room.LKRoom})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
