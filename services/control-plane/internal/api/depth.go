package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"era/services/control-plane/internal/rbac"
)

func (s *Server) handleAssetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !rbac.CanReadCases(rbac.FromRequest(r)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/assets/")
	id = strings.Trim(id, "/")
	if id == "" || id == "register" {
		http.NotFound(w, r)
		return
	}
	st := s.scopedStore(r)
	a, ok := st.GetAsset(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleCaseBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !rbac.CanReadCases(rbac.FromRequest(r)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	caseID := r.URL.Query().Get("case_id")
	if caseID == "" {
		http.Error(w, "case_id required", http.StatusBadRequest)
		return
	}
	c, ok := s.Store.GetCase(caseID)
	if !ok {
		http.Error(w, "case not found", http.StatusNotFound)
		return
	}
	out := map[string]any{
		"case":     c,
		"timeline": []any{},
		"exposure": nil,
	}
	// Best-effort timeline via event-writer.
	if c.NodeID != "" {
		base := serviceURL("ERA_EVENT_WRITER_URL", "http://127.0.0.1:8089")
		resp, err := http.Get(base + "/api/timeline?node_id=" + c.NodeID + "&limit=50")
		if err == nil {
			defer resp.Body.Close()
			var body any
			if json.NewDecoder(resp.Body).Decode(&body) == nil {
				out["timeline"] = body
			}
		}
		det := serviceURL("ERA_DETECTION_URL", "http://127.0.0.1:8097")
		er, err := http.Get(det + "/api/v1/exposure?asset_id=" + c.NodeID)
		if err == nil {
			defer er.Body.Close()
			var body any
			if json.NewDecoder(er.Body).Decode(&body) == nil {
				out["exposure"] = body
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleExposureByAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !rbac.CanReadCases(rbac.FromRequest(r)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/exposure/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	base := serviceURL("ERA_DETECTION_URL", "http://127.0.0.1:8097")
	target := base + "/api/v1/exposure?asset_id=" + id
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

// BYO connector admin (lab in-memory).
type byoConnector struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Source string `json:"source_type"`
	Config string `json:"config,omitempty"`
}

var (
	byoMu   sync.Mutex
	byoStore = map[string]*byoConnector{
		"default": {ID: "default", Name: "generic-json", Status: "ready", Source: "era.byo-edr.generic"},
	}
)

func (s *Server) handleBYOConnectors(w http.ResponseWriter, r *http.Request) {
	role := rbac.FromRequest(r)
	switch r.Method {
	case http.MethodGet:
		if !rbac.CanReadCases(role) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		byoMu.Lock()
		defer byoMu.Unlock()
		list := make([]*byoConnector, 0, len(byoStore))
		for _, c := range byoStore {
			list = append(list, c)
		}
		writeJSON(w, http.StatusOK, map[string]any{"connectors": list})
	case http.MethodPost:
		if !rbac.CanWriteCases(role) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var c byoConnector
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if c.ID == "" {
			c.ID = c.Name
		}
		if c.ID == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		if c.Status == "" {
			c.Status = "ready"
		}
		if c.Source == "" {
			c.Source = "era.byo-edr.generic"
		}
		byoMu.Lock()
		byoStore[c.ID] = &c
		byoMu.Unlock()
		writeJSON(w, http.StatusOK, c)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBYOConnectorSub(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/byo/connectors/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	role := rbac.FromRequest(r)
	switch r.Method {
	case http.MethodGet:
		if !rbac.CanReadCases(role) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		byoMu.Lock()
		c, ok := byoStore[id]
		byoMu.Unlock()
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, c)
	case http.MethodDelete:
		if !rbac.IsAdmin(role) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		byoMu.Lock()
		delete(byoStore, id)
		byoMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
