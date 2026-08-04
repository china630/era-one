package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"era/services/control-plane/internal/store"
)

func (s *Server) handleSuppressions(w http.ResponseWriter, r *http.Request) {
	if s.Suppressions == nil {
		s.Suppressions = store.NewSuppressionMem()
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"suppressions": s.Suppressions.List()})
	case http.MethodPost:
		var req struct {
			TenantID  string `json:"tenant_id"`
			RuleID    string `json:"rule_id"`
			NodeID    string `json:"node_id"`
			Reason    string `json:"reason"`
			ExpiresIn int    `json:"expires_in_hours"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RuleID == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		sp := &store.Suppression{
			TenantID:  req.TenantID,
			RuleID:    req.RuleID,
			NodeID:    req.NodeID,
			Reason:    req.Reason,
			CreatedBy: r.Header.Get("X-ERA-Actor"),
		}
		if req.ExpiresIn > 0 {
			t := time.Now().UTC().Add(time.Duration(req.ExpiresIn) * time.Hour)
			sp.ExpiresAt = &t
		}
		out := s.Suppressions.Create(sp)
		writeJSON(w, http.StatusCreated, out)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSuppressionSub(w http.ResponseWriter, r *http.Request) {
	if s.Suppressions == nil {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/suppressions/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Suppressions.Delete(id) {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}
