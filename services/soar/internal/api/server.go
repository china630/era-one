package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"era/services/platform/licensegate"
	"era/services/soar/internal/playbooks"
)

// PlaybookInfo — запись каталога известных плейбуков.
type PlaybookInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var playbookCatalog = []PlaybookInfo{
	{Name: "isolate_host", Description: "Isolate a host via firewall / agent quarantine"},
	{Name: "block_ip", Description: "Block an IP at the perimeter"},
	{Name: "create_ticket", Description: "Create an incident ticket linked to a case"},
}

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
	mux.HandleFunc("/api/v1/playbooks", s.handlePlaybooksCatalog)
	mux.HandleFunc("/api/v1/playbooks/", s.handlePlaybook)
	mux.HandleFunc("/api/v1/actions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"actions": s.Eng.Actions()})
	})
	mux.HandleFunc("/api/v1/actions/", s.handleAction)
	return mux
}

func (s *Server) handlePlaybooksCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"playbooks": playbookCatalog})
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/actions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		a, ok := s.Eng.ActionByID(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, a)
		return
	}
	if len(parts) == 2 && parts[1] == "retry" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !s.Gate.Allow(licensegate.ModuleResponse) {
			http.Error(w, "module response not licensed", http.StatusForbidden)
			return
		}
		a, ok := s.Eng.Retry(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, a)
		return
	}
	http.NotFound(w, r)
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
