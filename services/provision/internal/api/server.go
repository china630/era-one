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
	"era/services/provision/internal/store"
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
	mux.HandleFunc("/api/v1/images", s.handleImages)
	mux.HandleFunc("/api/v1/images/", s.handleImageDetail)
	mux.HandleFunc("/api/v1/pxe/config", s.handlePXE)
	mux.HandleFunc("/api/v1/enroll/jobs", s.handleEnrollJobs)
	mux.HandleFunc("/api/v1/enroll", s.handleEnroll)
	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir(uiDir()))))
	return mux
}

func uiDir() string {
	if d := os.Getenv("ERA_UI_DIR"); d != "" {
		return d
	}
	candidates := []string{"../../ui/provision", "ui/provision"}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs
			}
			return c
		}
	}
	return "../../ui/provision"
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
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"images": s.Store.ListImages()})
	case http.MethodPost:
		var img store.OSImage
		if err := json.NewDecoder(r.Body).Decode(&img); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if img.Name == "" || img.Platform == "" {
			http.Error(w, "name and platform required", http.StatusBadRequest)
			return
		}
		if img.ID == "" {
			img.ID = "img-" + uuid.NewString()[:8]
		}
		if img.CreatedAt.IsZero() {
			img.CreatedAt = time.Now().UTC()
		}
		if err := s.Store.CreateImage(&img); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, http.StatusCreated, img)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleImageDetail(w http.ResponseWriter, r *http.Request) {
	if !s.requireProvision(w) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/images/"), "/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		img, ok := s.Store.GetImage(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, img)
	case http.MethodDelete:
		if !s.Store.DeleteImage(id) {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePXE(w http.ResponseWriter, r *http.Request) {
	if !s.requireProvision(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.Store.PXEConfig())
	case http.MethodPut:
		var cfg store.PXEConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if cfg.TFTPRoot == "" {
			http.Error(w, "tftp_root required", http.StatusBadRequest)
			return
		}
		s.Store.SetPXEConfig(cfg)
		writeJSON(w, http.StatusOK, s.Store.PXEConfig())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleEnrollJobs(w http.ResponseWriter, r *http.Request) {
	if !s.requireProvision(w) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": s.Store.ListEnrollJobs()})
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
	job := &store.EnrollJob{
		ID:       uuid.NewString(),
		NodeID:   req.NodeID,
		Hostname: req.Hostname,
		AgentID:  req.AgentID,
		ImageID:  req.ImageID,
	}
	if s.CP == nil {
		job.Status = "failed"
		job.Error = "control-plane not configured"
		s.Store.RecordEnrollJob(job)
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
		job.Status = "failed"
		job.Error = err.Error()
		s.Store.RecordEnrollJob(job)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	job.Status = "enrolled"
	s.Store.RecordEnrollJob(job)
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "enrolled",
		"node_id": req.NodeID,
		"job_id":  job.ID,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
