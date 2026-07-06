package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"era/services/platform/licensegate"
	"era/services/soar/internal/playbooks"
)

type Server struct {
	Eng  *playbooks.Engine
	Gate *licensegate.Gate
}

func New(eng *playbooks.Engine, gate *licensegate.Gate) *Server {
	return &Server{Eng: eng, Gate: gate}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/playbooks/", s.handlePlaybook)
	mux.HandleFunc("/api/v1/actions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"actions": s.Eng.Actions()})
	})
	return mux
}

func (s *Server) handlePlaybook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleResponse) {
		http.Error(w, "module response not licensed", http.StatusForbidden)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/playbooks/")
	switch name {
	case "isolate_host":
		var req struct {
			NodeID string `json:"node_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, s.Eng.IsolateHost(req.NodeID))
	case "block_ip":
		var req struct {
			IP string `json:"ip"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IP == "" {
			http.Error(w, "ip required", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, s.Eng.BlockIP(req.IP))
	case "create_ticket":
		var req struct {
			Title  string `json:"title"`
			CaseID string `json:"case_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, s.Eng.CreateTicket(req.Title, req.CaseID))
	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
