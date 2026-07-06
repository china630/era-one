package api

import (
	"encoding/json"
	"net/http"

	"era/services/control-plane/internal/rbac"
	"era/services/control-plane/internal/store"
)

func (s *Server) handleHybridStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pol := s.Store.GetHybridPolicy()
	rt := s.Store.GetHybridRuntime()
	writeJSON(w, http.StatusOK, map[string]any{
		"connected": pol.Enabled,
		"policy":    pol,
		"runtime":   rt,
		"egress":    s.Store.ListEgressAudit(50),
	})
}

func (s *Server) handleHybridPolicy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"policy": s.Store.GetHybridPolicy()})
	case http.MethodPut:
		if rbac.FromRequest(r) != rbac.RoleAdmin {
			http.Error(w, "admin only", http.StatusForbidden)
			return
		}
		var pol store.HybridPolicy
		if err := json.NewDecoder(r.Body).Decode(&pol); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		prev := s.Store.GetHybridPolicy()
		s.Store.SetHybridPolicy(pol)
		s.Store.RecordAudit("hybrid.policy_update", s.actor(r), pol.DeploymentID,
			"enabled="+boolStr(pol.Enabled))
		if prev.Enabled != pol.Enabled {
			s.Store.RecordAudit("hybrid.connected_toggle", s.actor(r), pol.DeploymentID,
				boolStr(pol.Enabled))
		}
		writeJSON(w, http.StatusOK, map[string]any{"policy": pol})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
