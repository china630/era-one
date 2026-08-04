package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"era/services/control-plane/internal/store"
	"github.com/google/uuid"
)

func (s *Server) mountManageDeploy(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/manage/deploy/jobs", s.handleDeployJobs)
	mux.HandleFunc("/api/v1/manage/deploy/jobs/", s.handleDeployJobSub)
	mux.HandleFunc("/api/v1/manage/deploy/packages", s.handleDeployPackages)
	mux.HandleFunc("/api/v1/manage/patch/plan", s.handlePatchPlan)
	mux.HandleFunc("/api/v1/manage/patch/jobs", s.handlePatchJobs)
}

// seedDeployPackages — in-memory air-gap package catalog for lab UI.
func seedDeployPackages() []store.DeployPackage {
	return []store.DeployPackage{
		{ID: "agent-win-x64", Name: "ERA Agent", Version: "1.0.0", PackageRef: "s3://era-packages/agent-win-x64.msi", Platform: "windows"},
		{ID: "agent-linux-amd64", Name: "ERA Agent", Version: "1.0.0", PackageRef: "s3://era-packages/agent-linux-amd64.deb", Platform: "linux"},
		{ID: "sensor-win-x64", Name: "ERA Sensor", Version: "1.0.0", PackageRef: "s3://era-packages/sensor-win-x64.msi", Platform: "windows"},
	}
}

func (s *Server) handleDeployPackages(w http.ResponseWriter, r *http.Request) {
	if !s.requireManage(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"packages": seedDeployPackages()})
}

func (s *Server) handleDeployJobs(w http.ResponseWriter, r *http.Request) {
	st := s.scopedStore(r)
	switch r.Method {
	case http.MethodGet:
		if !s.requireManage(w, r) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"jobs": st.ListDeployJobs()})
	case http.MethodPost:
		if !s.requireManageAdmin(w, r) {
			return
		}
		var req struct {
			NodeID     string `json:"node_id"`
			TenantID   string `json:"tenant_id"`
			PackageRef string `json:"package_ref"`
			OTAToken   string `json:"ota_token"`
			Reboot     bool   `json:"reboot"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeID == "" || req.PackageRef == "" {
			http.Error(w, "node_id and package_ref required", http.StatusBadRequest)
			return
		}
		job := &store.DeployJob{
			ID: uuid.NewString(), TenantID: req.TenantID, NodeID: req.NodeID,
			PackageRef: req.PackageRef, OTAToken: req.OTAToken, Reboot: req.Reboot,
			Status: store.RolloutPending,
		}
		st.CreateDeployJob(job)
		st.RecordAudit("manage.deploy", s.actor(r), job.NodeID, job.PackageRef)
		writeJSON(w, http.StatusCreated, job)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDeployJobSub(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/manage/deploy/jobs/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	st := s.scopedStore(r)
	switch r.Method {
	case http.MethodGet:
		if !s.requireManage(w, r) {
			return
		}
		job, ok := st.GetDeployJob(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, job)
	case http.MethodPatch:
		if !s.requireManageAdmin(w, r) {
			return
		}
		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" {
			http.Error(w, "status required", http.StatusBadRequest)
			return
		}
		job, ok := st.UpdateDeployJob(id, store.RolloutStatus(req.Status))
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, job)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePatchPlan(w http.ResponseWriter, r *http.Request) {
	if !s.requireManage(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := s.scopedStore(r)
	plan := st.PlanPatches(nil)
	writeJSON(w, http.StatusOK, map[string]any{"plan": plan})
}

func (s *Server) handlePatchJobs(w http.ResponseWriter, r *http.Request) {
	st := s.scopedStore(r)
	switch r.Method {
	case http.MethodGet:
		if !s.requireManage(w, r) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"jobs": st.ListPatchJobs()})
	case http.MethodPost:
		if !s.requireManageAdmin(w, r) {
			return
		}
		var req struct {
			NodeID     string `json:"node_id"`
			TenantID   string `json:"tenant_id"`
			CVEID      string `json:"cve_id"`
			Product    string `json:"product"`
			PackageRef string `json:"package_ref"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeID == "" || req.PackageRef == "" {
			http.Error(w, "node_id and package_ref required", http.StatusBadRequest)
			return
		}
		job := &store.PatchJob{
			ID: uuid.NewString(), TenantID: req.TenantID, NodeID: req.NodeID,
			CVEID: req.CVEID, Product: req.Product, PackageRef: req.PackageRef,
			Status: store.RolloutPending,
		}
		st.CreatePatchJob(job)
		st.RecordAudit("manage.patch", s.actor(r), job.NodeID, job.CVEID)
		writeJSON(w, http.StatusCreated, job)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
