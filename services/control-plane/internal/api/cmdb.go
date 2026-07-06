package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"era/services/control-plane/internal/rbac"
	"era/services/control-plane/internal/store"
	"era/services/platform/licensegate"
)

func (s *Server) mountCMDB(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/cmdb/assets/", s.handleCMDBAsset)
	mux.HandleFunc("/api/v1/cmdb/software", s.handleCMDBSoftware)
	mux.HandleFunc("/api/v1/cmdb/contracts", s.handleCMDBContracts)
	mux.HandleFunc("/api/v1/cmdb/licenses", s.handleCMDBLicenses)
	mux.HandleFunc("/api/v1/cmdb/reconcile", s.handleCMDBReconcile)
}

func (s *Server) requireManage(w http.ResponseWriter, r *http.Request) bool {
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

func (s *Server) handleCMDBAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireManage(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/cmdb/assets/")
	if id == "" {
		http.Error(w, "node_id required", http.StatusBadRequest)
		return
	}
	st := s.scopedStore(r)
	a, ok := st.GetAsset(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	sw := st.ListAssetSoftware(id)
	writeJSON(w, http.StatusOK, map[string]any{"asset": a, "software": sw})
}

func (s *Server) handleCMDBSoftware(w http.ResponseWriter, r *http.Request) {
	if !s.requireManage(w, r) {
		return
	}
	st := s.scopedStore(r)
	switch r.Method {
	case http.MethodGet:
		nodeID := r.URL.Query().Get("node_id")
		var sw any
		if nodeID != "" {
			sw = st.ListAssetSoftware(nodeID)
		} else {
			sw = st.ListAllAssetSoftware()
		}
		writeJSON(w, http.StatusOK, map[string]any{"software": sw})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCMDBContracts(w http.ResponseWriter, r *http.Request) {
	if !s.requireManage(w, r) {
		return
	}
	st := s.scopedStore(r)
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"contracts": st.ListContracts()})
	case http.MethodPost:
		var c store.Contract
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		st.UpsertContract(&c)
		st.RecordAudit("cmdb.contract_create", s.actor(r), c.ID, c.Name)
		writeJSON(w, http.StatusCreated, c)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCMDBLicenses(w http.ResponseWriter, r *http.Request) {
	if !s.requireManage(w, r) {
		return
	}
	st := s.scopedStore(r)
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"licenses": st.ListSoftwareLicenses()})
	case http.MethodPost:
		var lic store.SoftwareLicense
		if err := json.NewDecoder(r.Body).Decode(&lic); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		st.UpsertSoftwareLicense(&lic)
		st.RecordAudit("cmdb.license_create", s.actor(r), lic.ID, lic.Product)
		writeJSON(w, http.StatusCreated, lic)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCMDBReconcile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireManage(w, r) {
		return
	}
	rows := s.scopedStore(r).ReconcileSoftwareLicenses()
	writeJSON(w, http.StatusOK, map[string]any{"reconcile": rows})
}
