package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"era/services/platform/cpclient"
	"era/services/platform/licensegate"
	"era/services/platform/metrics"
	"era/services/service-desk/internal/store"
	"github.com/google/uuid"
)

type Server struct {
	Store store.Repository
	Gate  *licensegate.Gate
	CP    *cpclient.Client
}

func New(st store.Repository, gate *licensegate.Gate, cp *cpclient.Client) *Server {
	return &Server{Store: st, Gate: gate, CP: cp}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/api/v1/incidents", s.handleIncidents)
	mux.HandleFunc("/api/v1/incidents/", s.handleIncidentSub)
	mux.HandleFunc("/api/v1/requests", s.handleRequests)
	mux.HandleFunc("/api/v1/problems", s.handleProblems)
	mux.HandleFunc("/api/v1/changes", s.handleChanges)
	mux.HandleFunc("/api/v1/cmdb/assets", s.handleCMDBAssets)
	return mux
}

func (s *Server) requireService(w http.ResponseWriter) bool {
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModuleService) {
		http.Error(w, "service module not licensed", http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) handleIncidents(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"incidents": s.Store.ListIncidents()})
	case http.MethodPost:
		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Priority    string `json:"priority"`
			NodeID      string `json:"node_id"`
			Requester   string `json:"requester"`
			TenantID    string `json:"tenant_id"`
			SLAHours    int    `json:"sla_hours"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		if req.NodeID != "" && s.CP != nil {
			if _, err := s.CP.GetAsset(req.NodeID); err != nil {
				http.Error(w, "node_id not in CMDB", http.StatusBadRequest)
				return
			}
		}
		inc := &store.Incident{
			ID:          uuid.NewString(),
			TenantID:    req.TenantID,
			Title:       req.Title,
			Description: req.Description,
			Priority:    req.Priority,
			NodeID:      req.NodeID,
			Requester:   req.Requester,
		}
		if req.SLAHours > 0 {
			due := time.Now().UTC().Add(time.Duration(req.SLAHours) * time.Hour)
			inc.SLADueAt = &due
		}
		s.Store.CreateIncident(inc)
		writeJSON(w, http.StatusCreated, inc)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleIncidentSub(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/incidents/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		inc, ok := s.Store.GetIncident(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, inc)
	case http.MethodPatch:
		var req struct {
			Status   string `json:"status"`
			Assignee string `json:"assignee"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		inc, ok := s.Store.UpdateIncident(id, func(i *store.Incident) {
			if req.Status != "" {
				i.Status = store.TicketStatus(req.Status)
			}
			if req.Assignee != "" {
				i.Assignee = req.Assignee
			}
		})
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, inc)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRequests(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"requests": s.Store.ListRequests()})
	case http.MethodPost:
		var req store.ServiceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.Requester == "" {
			http.Error(w, "title and requester required", http.StatusBadRequest)
			return
		}
		req.ID = uuid.NewString()
		s.Store.CreateRequest(&req)
		writeJSON(w, http.StatusCreated, req)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProblems(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"problems": s.Store.ListProblems()})
	case http.MethodPost:
		var p store.Problem
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.Title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		p.ID = uuid.NewString()
		s.Store.CreateProblem(&p)
		writeJSON(w, http.StatusCreated, p)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleChanges(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"changes": s.Store.ListChanges()})
	case http.MethodPost:
		var c store.Change
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil || c.Title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		c.ID = uuid.NewString()
		s.Store.CreateChange(&c)
		writeJSON(w, http.StatusCreated, c)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCMDBAssets(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.CP == nil {
		writeJSON(w, http.StatusOK, map[string]any{"assets": []any{}})
		return
	}
	assets, err := s.CP.ListAssets()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": assets})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func ListenAddr(addr string, h http.Handler) error {
	return http.ListenAndServe(addr, h)
}
