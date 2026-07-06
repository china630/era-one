package api

import (
	"io"
	"net/http"
	"net/url"

	"era/services/control-plane/internal/rbac"
)

func (s *Server) handleWorkbenchTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !rbac.CanReadCases(rbac.FromRequest(r)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	q := r.URL.Query()
	caseID := q.Get("case_id")
	nodeID := q.Get("node_id")
	corrID := q.Get("correlation_id")
	if caseID != "" && nodeID == "" {
		if c, ok := s.Store.GetCase(caseID); ok && c.NodeID != "" {
			nodeID = c.NodeID
		}
	}
	if nodeID == "" && corrID == "" {
		http.Error(w, "case_id, node_id or correlation_id required", http.StatusBadRequest)
		return
	}
	base := serviceURL("ERA_EVENT_WRITER_URL", "http://127.0.0.1:8089")
	params := url.Values{}
	if nodeID != "" {
		params.Set("node_id", nodeID)
	}
	if corrID != "" {
		params.Set("correlation_id", corrID)
	}
	if lim := q.Get("limit"); lim != "" {
		params.Set("limit", lim)
	}
	target := base + "/api/timeline?" + params.Encode()
	resp, err := http.Get(target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleExposureProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !rbac.CanReadCases(rbac.FromRequest(r)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	base := serviceURL("ERA_DETECTION_URL", "http://127.0.0.1:8097")
	target := base + "/api/v1/exposure"
	if q := r.URL.RawQuery; q != "" {
		target += "?" + q
	}
	resp, err := http.Get(target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
