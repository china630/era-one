package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
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
	mux.HandleFunc("/api/v1/requests/", s.handleRequestSub)
	mux.HandleFunc("/api/v1/problems", s.handleProblems)
	mux.HandleFunc("/api/v1/problems/", s.handleProblemSub)
	mux.HandleFunc("/api/v1/changes", s.handleChanges)
	mux.HandleFunc("/api/v1/changes/", s.handleChangeSub)
	mux.HandleFunc("/api/v1/cmdb/assets", s.handleCMDBAssets)
	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir(uiDir()))))
	return mux
}

func uiDir() string {
	if d := os.Getenv("ERA_UI_DIR"); d != "" {
		return d
	}
	// from services/service-desk when running via go test / go run
	candidates := []string{"../../ui/service-desk", "ui/service-desk"}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "../../ui/service-desk"
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
		list := s.Store.ListIncidents()
		for _, inc := range list {
			store.EnrichIncident(inc)
		}
		writeJSON(w, http.StatusOK, map[string]any{"incidents": list})
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
		writeJSON(w, http.StatusCreated, store.EnrichIncident(inc))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleIncidentSub(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/incidents/")
	parts := splitPath(rest)
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	id := parts[0]
	if len(parts) == 2 && parts[1] == "comments" {
		s.handleComments(w, r, store.KindIncident, id, func() bool {
			_, ok := s.Store.GetIncident(id)
			return ok
		})
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		inc, ok := s.Store.GetIncident(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, store.EnrichIncident(inc))
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
		writeJSON(w, http.StatusOK, store.EnrichIncident(inc))
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

func (s *Server) handleRequestSub(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/requests/")
	parts := splitPath(rest)
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	id := parts[0]
	if len(parts) == 2 && parts[1] == "comments" {
		s.handleComments(w, r, store.KindRequest, id, func() bool {
			_, ok := s.Store.GetRequest(id)
			return ok
		})
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		req, ok := s.Store.GetRequest(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, req)
	case http.MethodPatch:
		var body struct {
			Status   string `json:"status"`
			Assignee string `json:"assignee"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		req, ok := s.Store.UpdateRequest(id, func(x *store.ServiceRequest) {
			if body.Status != "" {
				x.Status = store.TicketStatus(body.Status)
			}
			if body.Assignee != "" {
				x.Assignee = body.Assignee
			}
		})
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, req)
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

func (s *Server) handleProblemSub(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/problems/"), "/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		p, ok := s.Store.GetProblem(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, p)
	case http.MethodPatch:
		var body struct {
			Status   string `json:"status"`
			Assignee string `json:"assignee"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		p, ok := s.Store.UpdateProblem(id, func(x *store.Problem) {
			if body.Status != "" {
				x.Status = store.TicketStatus(body.Status)
			}
			if body.Assignee != "" {
				x.Assignee = body.Assignee
			}
		})
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, p)
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

func (s *Server) handleChangeSub(w http.ResponseWriter, r *http.Request) {
	if !s.requireService(w) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/changes/"), "/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		c, ok := s.Store.GetChange(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, c)
	case http.MethodPatch:
		var body struct {
			Status   string `json:"status"`
			Assignee string `json:"assignee"`
			Risk     string `json:"risk"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		c, ok := s.Store.UpdateChange(id, func(x *store.Change) {
			if body.Status != "" {
				x.Status = store.TicketStatus(body.Status)
			}
			if body.Assignee != "" {
				x.Assignee = body.Assignee
			}
			if body.Risk != "" {
				x.Risk = body.Risk
			}
		})
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, c)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleComments(w http.ResponseWriter, r *http.Request, kind store.TicketKind, ticketID string, exists func() bool) {
	if !exists() {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"comments": s.Store.ListComments(kind, ticketID)})
	case http.MethodPost:
		var body struct {
			Author string `json:"author"`
			Body   string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Body == "" {
			http.Error(w, "body required", http.StatusBadRequest)
			return
		}
		if body.Author == "" {
			body.Author = "anonymous"
		}
		c := &store.Comment{
			ID:       uuid.NewString(),
			TicketID: ticketID,
			Kind:     kind,
			Author:   body.Author,
			Body:     body.Body,
		}
		s.Store.AddComment(c)
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

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func ListenAddr(addr string, h http.Handler) error {
	return http.ListenAndServe(addr, h)
}
