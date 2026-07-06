package api

import (
	"encoding/json"
	"io"
	"net/http"

	"era/services/federated/internal/hub"
	"era/services/platform/envelope"
	"era/services/platform/licensegate"
)

type Server struct {
	Hub  HubAPI
	Gate *licensegate.Gate
}

func New(h HubAPI, gate *licensegate.Gate) *Server {
	return &Server{Hub: h, Gate: gate}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/federated/submit", s.handleSubmit)
	mux.HandleFunc("/api/v1/federated/aggregate", s.handleAggregate)
	mux.HandleFunc("/api/v1/federated/model", s.handleModel)
	mux.HandleFunc("/api/v1/federated/audit", s.handleAudit)
	return mux
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleFederated) {
		http.Error(w, "module federated not licensed", http.StatusForbidden)
		return
	}
	if !hub.ValidateZoneToken(zoneToken(r)) {
		http.Error(w, "invalid zone token", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := envelope.ValidateNoPII(string(body)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var sub hub.GradientSubmission
	if err := json.Unmarshal(body, &sub); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.Hub.Submit(sub); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *Server) handleAggregate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleFederated) {
		http.Error(w, "module federated not licensed", http.StatusForbidden)
		return
	}
	model, round := s.Hub.Aggregate()
	writeJSON(w, http.StatusOK, map[string]any{"round": round, "model": model})
}

func (s *Server) handleModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleFederated) {
		http.Error(w, "module federated not licensed", http.StatusForbidden)
		return
	}
	model, round := s.Hub.GlobalModel()
	writeJSON(w, http.StatusOK, map[string]any{"round": round, "model": model})
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleFederated) {
		http.Error(w, "module federated not licensed", http.StatusForbidden)
		return
	}
	if !hub.ValidateZoneToken(zoneToken(r)) {
		http.Error(w, "invalid zone token", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": s.Hub.AuditEntries()})
}

func zoneToken(r *http.Request) string {
	if t := r.Header.Get("X-ERA-Zone-Token"); t != "" {
		return t
	}
	const prefix = "Bearer "
	if h := r.Header.Get("Authorization"); len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return ""
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
