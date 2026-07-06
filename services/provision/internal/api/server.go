package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"era/services/platform/cpclient"
	"era/services/platform/licensegate"
	"era/services/platform/metrics"
	"era/services/provision/internal/store"
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
	mux.HandleFunc("/api/v1/images", s.handleImages)
	mux.HandleFunc("/api/v1/images/", s.handleImageDetail)
	mux.HandleFunc("/api/v1/pxe/config", s.handlePXE)
	mux.HandleFunc("/api/v1/enroll", s.handleEnroll)
	mux.HandleFunc("/ui/", s.handleUI)
	return mux
}

func (s *Server) requireProvision(w http.ResponseWriter) bool {
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModuleProvision) {
		http.Error(w, "provision module not licensed", http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) handleImages(w http.ResponseWriter, r *http.Request) {
	if !s.requireProvision(w) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"images": s.Store.ListImages()})
}

func (s *Server) handleImageDetail(w http.ResponseWriter, r *http.Request) {
	if !s.requireProvision(w) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/images/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	img, ok := s.Store.GetImage(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, img)
}

func (s *Server) handlePXE(w http.ResponseWriter, r *http.Request) {
	if !s.requireProvision(w) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.Store.PXEConfig())
}

func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	if !s.requireProvision(w) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req store.EnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.NodeID == "" || req.Hostname == "" || req.AgentID == "" {
		http.Error(w, "node_id, hostname, agent_id required", http.StatusBadRequest)
		return
	}
	if s.CP == nil {
		http.Error(w, "control-plane not configured", http.StatusServiceUnavailable)
		return
	}
	if req.TenantID == "" {
		req.TenantID = "tenant-dev"
	}
	if req.Platform == "" {
		req.Platform = "linux"
	}
	if req.AgentVersion == "" {
		req.AgentVersion = "0.1.0"
	}
	if err := s.CP.WithActor("era-provision").RegisterAsset(
		req.AgentID, req.TenantID, req.NodeID, req.Hostname, req.Platform, req.AgentVersion,
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "enrolled",
		"node_id": req.NodeID,
	})
}

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/ui/" && r.URL.Path != "/ui" {
		http.NotFound(w, r)
		return
	}
	dir := os.Getenv("ERA_UI_DIR")
	if dir == "" {
		dir = "../../ui/provision"
	}
	http.Redirect(w, r, "/ui/provision/", http.StatusFound)
	_ = dir
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
