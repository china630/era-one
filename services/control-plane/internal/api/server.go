package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"era/services/control-plane/internal/rbac"
	"era/services/control-plane/internal/store"
	"era/services/platform/licensegate"
	"era/services/platform/metrics"
	"github.com/google/uuid"
)

type Server struct {
	Store store.Repository
	Gate  *licensegate.Gate
}

func New(st store.Repository, gate *licensegate.Gate) *Server {
	return &Server{Store: st, Gate: gate}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/api/v1/policy", s.handlePolicy)
	mux.HandleFunc("/api/v1/assets", s.handleAssets)
	mux.HandleFunc("/api/v1/assets/register", s.handleRegister)
	mux.HandleFunc("/api/v1/cases", s.handleCases)
	mux.HandleFunc("/api/v1/cases/", s.handleCaseSub)
	mux.HandleFunc("/api/v1/audit", s.handleAudit)
	mux.HandleFunc("/api/v1/hybrid/status", s.handleHybridStatus)
	mux.HandleFunc("/api/v1/hybrid/policy", s.handleHybridPolicy)
    mux.HandleFunc("/api/v1/license/modules", s.handleModules)
    s.mountProxies(mux)
    s.mountCMDB(mux)
    s.mountCMDBNetwork(mux)
    s.mountEnforcement(mux)
    s.mountManageDeploy(mux)
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/" || r.URL.Path == "/ui" || r.URL.Path == "/ui/" {
            http.Redirect(w, r, "/ui/portal/", http.StatusFound)
            return
        }
        http.NotFound(w, r)
    })
    mux.Handle("/ui/", noCacheUI(http.StripPrefix("/ui/", http.FileServer(http.Dir(uiDir())))))
	return rbac.Middleware(mux)
}

func (s *Server) actor(r *http.Request) string {
	if a := r.Header.Get("X-ERA-Actor"); a != "" {
		return a
	}
	return string(rbac.FromRequest(r))
}

func (s *Server) scopedStore(r *http.Request) store.Repository {
	return store.Scoped(s.Store, r.Header.Get("X-ERA-Tenant-ID"))
}

func (s *Server) handlePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.Store.Policy())
}

type registerReq struct {
	AgentID      string `json:"agent_id"`
	TenantID     string `json:"tenant_id"`
	NodeID       string `json:"node_id"`
	Hostname     string `json:"hostname"`
	Platform     string `json:"platform"`
	AgentVersion string `json:"agent_version"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.Store.UpsertAsset(&store.Asset{
		NodeID: req.NodeID, TenantID: req.TenantID, Hostname: req.Hostname,
		Platform: req.Platform, AgentID: req.AgentID, AgentVersion: req.AgentVersion,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"policy_version": s.Store.Policy().Version,
		"coverage":       s.Store.AssetCoverage(),
	})
}

func (s *Server) handleAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := s.scopedStore(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"assets": st.ListAssets(), "coverage": st.AssetCoverage(),
	})
}

func (s *Server) handleCases(w http.ResponseWriter, r *http.Request) {
	role := rbac.FromRequest(r)
	switch r.Method {
	case http.MethodGet:
		if !rbac.CanReadCases(role) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		status := r.URL.Query().Get("status")
		cases := s.scopedStore(r).ListCases()
		if status != "" {
			filtered := cases[:0]
			for _, c := range cases {
				if c.Status == status {
					filtered = append(filtered, c)
				}
			}
			cases = filtered
		}
		writeJSON(w, http.StatusOK, map[string]any{"cases": cases})
	case http.MethodPost:
		if !rbac.CanWriteCases(role) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var req struct {
			Title       string `json:"title"`
			TenantID     string `json:"tenant_id"`
			DetectionID string `json:"detection_id"`
			NodeID      string `json:"node_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		tenantID := req.TenantID
		if tenantID == "" {
			tenantID = r.Header.Get("X-ERA-Tenant-ID")
		}
		c := &store.Case{
			ID: uuid.NewString(), Title: req.Title, Status: "new", TenantID: tenantID,
			DetectionID: req.DetectionID, NodeID: req.NodeID,
		}
		s.Store.CreateCase(c)
		s.Store.RecordAudit("case.create", s.actor(r), c.ID, req.Title)
		writeJSON(w, http.StatusCreated, c)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCaseSub(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/cases/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 2 && parts[1] == "notes" && r.Method == http.MethodPost {
		s.handleAddNote(w, r, id)
		return
	}
	switch r.Method {
	case http.MethodGet:
		d, ok := s.Store.GetCaseDetail(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, d)
	case http.MethodPatch:
		if !rbac.CanWriteCases(rbac.FromRequest(r)) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var req struct {
			Status   string `json:"status"`
			Assignee string `json:"assignee"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		c, ok := s.Store.UpdateCase(id, func(c *store.Case) {
			if req.Status != "" {
				c.Status = req.Status
			}
			if req.Assignee != "" {
				c.Assignee = req.Assignee
			}
		})
		if !ok {
			http.NotFound(w, r)
			return
		}
		if req.Assignee != "" {
			s.Store.AddTimeline(id, "assign", s.actor(r), "assigned to "+req.Assignee)
		}
		if req.Status != "" {
			s.Store.AddTimeline(id, "status", s.actor(r), "status → "+req.Status)
			s.Store.RecordAudit("case.status", s.actor(r), id, req.Status)
		}
		writeJSON(w, http.StatusOK, c)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAddNote(w http.ResponseWriter, r *http.Request, caseID string) {
	if !rbac.CanWriteCases(rbac.FromRequest(r)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Body == "" {
		http.Error(w, "body required", http.StatusBadRequest)
		return
	}
	n, ok := s.Store.AddCaseNote(caseID, s.actor(r), req.Body)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.Store.AddTimeline(caseID, "note", s.actor(r), "note added")
	writeJSON(w, http.StatusCreated, n)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if rbac.FromRequest(r) != rbac.RoleAdmin {
		http.Error(w, "admin only", http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"audit": s.Store.ListAudit(200)})
}

func (s *Server) handleModules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mods := map[string]bool{}
	for _, m := range licensegate.KnownModules {
		mods[string(m)] = s.Gate.Allow(m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"modules": mods})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func uiDir() string {
	if p := os.Getenv("ERA_UI_DIR"); p != "" {
		return p
	}
	return "../../ui"
}

// CreateCaseFromExternal — для SOAR/AI интеграции (S5-26, S5-27).
func CreateCaseFromExternal(st store.Repository, title, detectionID, nodeID, actor string) *store.Case {
	c := &store.Case{
		ID: uuid.NewString(), Title: title, Status: "new",
		DetectionID: detectionID, NodeID: nodeID,
	}
	st.CreateCase(c)
	st.RecordAudit("case.auto_create", actor, c.ID, title)
	return c
}
