package api

import (
	"encoding/json"
	"net/http"
	"os"

	"era/services/ai-core/internal/investigate"
	"era/services/platform/cpclient"
	"era/services/platform/licensegate"
)

type Server struct {
	Inv *investigate.Client
	Gate *licensegate.Gate
	CP  *cpclient.Client
}

func New(inv *investigate.Client, gate *licensegate.Gate) *Server {
	return &Server{
		Inv: inv, Gate: gate,
		CP: cpclient.New(os.Getenv("ERA_CONTROL_PLANE_URL")).WithActor("ai-core"),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/investigate", s.handleInvestigate)
	return mux
}

func (s *Server) handleInvestigate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleControlAI) {
		http.Error(w, "module ai not licensed", http.StatusForbidden)
		return
	}
	var req investigate.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	res, err := s.Inv.Investigate(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.CP != nil && (res.Verdict == "malicious" || res.Verdict == "suspicious") {
		title := "AI investigation: " + res.Verdict + " on " + req.NodeID
		if id, err := s.CP.CreateCase(title, req.DetectionID, req.NodeID); err == nil {
			res.CaseID = id
		}
	}
	writeJSON(w, http.StatusOK, res)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
