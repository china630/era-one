package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"era/services/control-plane/internal/networkreconcile"
	"era/services/control-plane/internal/rbac"
	"era/services/platform/licensegate"
)

func (s *Server) mountCMDBNetwork(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/cmdb/network/assets", s.handleCMDBNetworkAssets)
}

func (s *Server) requireObserve(w http.ResponseWriter, r *http.Request) bool {
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModuleObserve) {
		http.Error(w, "observe module not licensed", http.StatusForbidden)
		return false
	}
	if !rbac.CanReadCases(rbac.FromRequest(r)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) handleCMDBNetworkAssets(w http.ResponseWriter, r *http.Request) {
	if !s.requireObserve(w, r) {
		return
	}
	st := s.scopedStore(r)
	switch r.Method {
	case http.MethodGet:
		assets := networkreconcile.ListNetwork(st)
		writeJSON(w, http.StatusOK, map[string]any{"assets": assets})
	case http.MethodPost:
		var in networkreconcile.Input
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if in.TenantID == "" {
			in.TenantID = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		}
		res := networkreconcile.Upsert(st, in)
		if res.Audit != "" && res.Conflict {
			st.RecordAudit("cmdb.observe_conflict", s.actor(r), in.NodeID, res.Audit)
		} else if res.Asset != nil {
			st.RecordAudit("cmdb.observe_upsert", s.actor(r), res.Asset.NodeID, res.Asset.Hostname)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"asset": res.Asset, "conflict": res.Conflict, "audit": res.Audit,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
