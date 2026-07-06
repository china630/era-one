package api

import (
	"encoding/json"
	"net/http"

	"era/services/dlp/internal/session"
	"era/services/platform/envelope"
)

type Server struct {
	Store *session.Store
	Pub   *envelope.Publisher
}

func New(st *session.Store, pub *envelope.Publisher) *Server {
	return &Server{Store: st, Pub: pub}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/sessions/start", s.handleStart)
	mux.HandleFunc("/api/v1/sessions/command", s.handleCommand)
	mux.HandleFunc("/api/v1/sessions/end", s.handleEnd)
	mux.HandleFunc("/api/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"sessions": s.Store.List(),
			"alerts":   s.Store.Alerts(),
		})
	})
	return mux
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		User string `json:"user"`
		Host string `json:"host"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	rec := s.Store.Start(req.User, req.Host)
	if s.Pub != nil {
		_ = s.Pub.PublishAuth(r.Context(), req.User, "privileged_session_start", true, req.Host)
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
		Command   string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	alert, fired := s.Store.LogCommand(req.SessionID, req.Command)
	if fired && s.Pub != nil {
		_ = s.Pub.PublishAuth(r.Context(), "privileged", "dlp_alert", false, req.Command)
	}
	writeJSON(w, http.StatusOK, map[string]any{"alert": alert, "fired": fired})
}

func (s *Server) handleEnd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	rec, ok := s.Store.End(req.SessionID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
