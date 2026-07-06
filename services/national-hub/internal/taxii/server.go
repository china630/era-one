package taxii

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"era/services/national-hub/internal/hub"
	"era/services/national-hub/internal/sanitize"
	"era/services/platform/licensegate"
	"github.com/google/uuid"
)

type Server struct {
	Store hub.ObjectStore
	Gate  *licensegate.Gate
}

func New(st hub.ObjectStore, gate *licensegate.Gate) *Server {
	return &Server{Store: st, Gate: gate}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/taxii2/", s.discovery)
	mux.HandleFunc("/taxii2/api1/collections/", s.collections)
	mux.HandleFunc("/taxii2/api1/collections/", s.collectionObjects)
	return mux
}

func (s *Server) RoutesWithObjects() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/taxii2/", s.discovery)
	mux.Handle("/taxii2/api1/collections/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/objects/") || strings.Contains(r.URL.Path, "/objects") {
			s.handleObjects(w, r)
			return
		}
		s.listCollections(w, r)
	}))
	return mux
}

func (s *Server) discovery(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/taxii2/" && r.URL.Path != "/taxii2" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"title": "ERA XDR National TAXII Hub",
		"api_roots": []string{"/taxii2/api1/"},
	})
}

func (s *Server) collections(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "objects") {
		s.handleObjects(w, r)
		return
	}
	s.listCollections(w, r)
}

func (s *Server) collectionObjects(w http.ResponseWriter, r *http.Request) {
	s.handleObjects(w, r)
}

func (s *Server) listCollections(w http.ResponseWriter, r *http.Request) {
	if !s.requireNational(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"collections": []map[string]any{{
			"id": hub.DefaultCollection, "title": "ERA National Threats",
			"can_read": true, "can_write": true,
		}},
	})
}

func (s *Server) handleObjects(w http.ResponseWriter, r *http.Request) {
	if !s.requireNational(w, r) {
		return
	}
	collection := hub.DefaultCollection
	orgID := r.Header.Get("X-ERA-Org-ID")
	if orgID == "" {
		orgID = "org-unknown"
	}

	switch r.Method {
	case http.MethodGet:
		objs := s.Store.Poll(collection)
		writeJSON(w, http.StatusOK, map[string]any{
			"objects": objs,
			"more":    false,
			"count":   s.Store.NoisyPublishCount(collection, 2.0),
		})
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := sanitize.AuditBundle(body); err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		id := uuid.NewString()
		s.Store.Publish(collection, orgID, id, body)
		writeJSON(w, http.StatusCreated, map[string]string{"id": id, "status": "published"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) requireNational(w http.ResponseWriter, r *http.Request) bool {
	if s.Gate.Allow(licensegate.ModuleNational) {
		return true
	}
	http.Error(w, "module national not licensed", http.StatusForbidden)
	return false
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
