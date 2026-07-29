package chat

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct{}

func NewServer() *Server { return &Server{} }

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/chat/healthz", s.healthz)
	mux.HandleFunc("/chat", s.withRBAC(s.index))
}

func (s *Server) withRBAC(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-ERA-Tenant") == "" || !strings.Contains(r.Header.Get("X-ERA-Role"), "chat.user") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "service": "ui-chat"})
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"route":   "/chat",
		"tenant":  r.Header.Get("X-ERA-Tenant"),
		"feature": "chat-shell",
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
