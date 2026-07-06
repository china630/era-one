package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"era/services/control-plane/internal/rbac"
	"era/services/control-plane/internal/store"
	"era/services/platform/licensegate"
)

func (s *Server) mountEnforcement(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/enforcement/policy", s.handleEnforcementPolicy)
	mux.HandleFunc("/api/v1/enforcement/virtual-patch", s.handleEnforcementVirtualPatch)
	mux.HandleFunc("/api/v1/enforcement/rollback", s.handleEnforcementRollback)
	mux.HandleFunc("/api/v1/enforcement/history", s.handleEnforcementHistory)
	mux.HandleFunc("/api/v1/enforcement/escrow", s.handleEnforcementEscrow)
	mux.HandleFunc("/api/v1/enforcement/escrow/", s.handleEnforcementEscrowDetail)
}

func (s *Server) handleEnforcementPolicy(w http.ResponseWriter, r *http.Request) {
	st := s.scopedStore(r)
	switch r.Method {
	case http.MethodGet:
		if !s.allowEnforcementRead(w, r) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"policy": st.GetEnforcementPolicy()})
	case http.MethodPut:
		if !s.requireManage(w, r) {
			return
		}
		var p store.EnforcementPolicy
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid policy json", http.StatusBadRequest)
			return
		}
		if err := st.SetEnforcementPolicy(p, s.actor(r), "policy update"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		st.RecordAudit("enforcement.policy", s.actor(r), p.Version, p.Mode)
		writeJSON(w, http.StatusOK, map[string]any{"policy": st.GetEnforcementPolicy()})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleEnforcementRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireManage(w, r) {
		return
	}
	st := s.scopedStore(r)
	p, ok := st.RollbackEnforcementPolicy(s.actor(r))
	if !ok {
		http.Error(w, "no previous policy", http.StatusConflict)
		return
	}
	st.RecordAudit("enforcement.rollback", s.actor(r), p.Version, p.Mode)
	writeJSON(w, http.StatusOK, map[string]any{"policy": p})
}

func (s *Server) handleEnforcementHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireManage(w, r) {
		return
	}
	st := s.scopedStore(r)
	writeJSON(w, http.StatusOK, map[string]any{"history": st.ListEnforcementHistory(50)})
}

func (s *Server) handleEnforcementEscrow(w http.ResponseWriter, r *http.Request) {
	if !s.requireManage(w, r) {
		return
	}
	st := s.scopedStore(r)
	switch r.Method {
	case http.MethodGet:
		nodeID := r.URL.Query().Get("node_id")
		writeJSON(w, http.StatusOK, map[string]any{"escrows": st.ListBitlockerEscrows(nodeID)})
	case http.MethodPost:
		var req struct {
			NodeID   string `json:"node_id"`
			TenantID string `json:"tenant_id"`
			VolumeID string `json:"volume_id"`
			KeyBlob  string `json:"key_blob"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.NodeID == "" || req.VolumeID == "" || req.KeyBlob == "" {
			http.Error(w, "node_id, volume_id, key_blob required", http.StatusBadRequest)
			return
		}
		st.UpsertBitlockerEscrow(&store.BitlockerEscrow{
			NodeID:   req.NodeID,
			TenantID: req.TenantID,
			VolumeID: req.VolumeID,
			KeyBlob:  req.KeyBlob,
			Actor:    s.actor(r),
		})
		st.RecordAudit("enforcement.escrow", s.actor(r), req.NodeID, req.VolumeID)
		writeJSON(w, http.StatusCreated, map[string]string{"status": "stored"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleEnforcementEscrowDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireManage(w, r) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/enforcement/escrow/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "node_id/volume_id required", http.StatusBadRequest)
		return
	}
	st := s.scopedStore(r)
	e, ok := st.GetBitlockerEscrow(parts[0], parts[1])
	if !ok {
		http.NotFound(w, r)
		return
	}
	if rbac.FromRequest(r) != rbac.RoleAdmin {
		pub := e.Public()
		writeJSON(w, http.StatusOK, pub)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) handleEnforcementVirtualPatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireManage(w, r) {
		return
	}
	var req struct {
		CVEID      string `json:"cve_id"`
		HashSHA256 string `json:"hash_sha256"`
		Path       string `json:"path"`
		Vector     string `json:"vector"`
		Mode       string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	st := s.scopedStore(r)
	cur := st.GetEnforcementPolicy()
	merged, err := store.MergeVirtualPatch(cur, req.CVEID, req.HashSHA256, req.Path, req.Vector)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	merged.Version = bumpPatchVersion(cur.Version)
	if req.Mode != "" {
		merged.Mode = req.Mode
	} else {
		merged.Mode = cur.Mode
	}
	merged.FailMode = cur.FailMode
	if err := st.SetEnforcementPolicy(merged, s.actor(r), "virtual patch "+req.CVEID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st.RecordAudit("enforcement.virtual_patch", s.actor(r), req.CVEID, req.HashSHA256)
	writeJSON(w, http.StatusOK, map[string]any{"policy": st.GetEnforcementPolicy()})
}

func bumpPatchVersion(cur string) string {
	if cur == "" {
		return "1.0.1-vp"
	}
	return cur + "+vp"
}

func (s *Server) allowEnforcementRead(w http.ResponseWriter, r *http.Request) bool {
	actor := r.Header.Get("X-ERA-Actor")
	if actor == "era-agent" {
		return true
	}
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModuleManage) {
		http.Error(w, "manage module not licensed", http.StatusForbidden)
		return false
	}
	if !rbac.CanReadCases(rbac.FromRequest(r)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}
