package api

import (
	"encoding/json"
	"net/http"

	"era/services/ngfw/internal/policy"
	"era/services/platform/envelope"
)

type Server struct {
	Engine *policy.Engine
	Pub    *envelope.Publisher
}

func New(eng *policy.Engine, pub *envelope.Publisher) *Server {
	return &Server{Engine: eng, Pub: pub}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/ngfw/evaluate", s.handleEvaluate)
	mux.HandleFunc("/api/v1/ngfw/policies", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"engine": "era-ngfw", "rules": s.Engine.Rules})
	})
	return mux
}

func (s *Server) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var flow policy.Flow
	if err := json.NewDecoder(r.Body).Decode(&flow); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	dec := s.Engine.Evaluate(flow.SrcIP, flow.DstIP, flow.Protocol, flow.DstPort)
	if s.Pub != nil {
		dir := "outbound"
		if !dec.Allowed {
			dir = "blocked"
		}
		_ = s.Pub.PublishNetwork(r.Context(), flow.SrcIP, flow.DstIP, flow.Protocol, dir, flow.DstPort)
	}
	writeJSON(w, http.StatusOK, map[string]any{"decision": dec, "flow": flow.String()})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
