package api

import (
	"encoding/json"
	"net/http"

	"era/services/comms/mail-connect/internal/audit"
	"era/services/comms/mail-connect/internal/autodiscover"
	syncstore "era/services/comms/mail-connect/internal/sync"
)

type Server struct {
	Store   *syncstore.Store
	Auditor *audit.Recorder
}

func NewServer(store *syncstore.Store, auditor *audit.Recorder) *Server {
	return &Server{Store: store, Auditor: auditor}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/v1/connect/mailboxes", s.registerMailbox)
	mux.HandleFunc("/api/v1/connect/sync", s.syncMailbox)
	mux.HandleFunc("/api/v1/connect/autodiscover.xml", s.connectAutodiscover)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "service": "era-mail-connect"})
}

func (s *Server) registerMailbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var m syncstore.Mailbox
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil || m.TenantID == "" || m.Email == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	s.Store.PutMailbox(m)
	s.Auditor.Record(audit.Event{Action: "CONNECT_REGISTER", TenantID: m.TenantID, Mailbox: m.Email})
	writeJSON(w, map[string]string{"status": "registered"})
}

func (s *Server) syncMailbox(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var payload struct {
			TenantID string `json:"tenant_id"`
			Mailbox  string `json:"mailbox"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.TenantID == "" || payload.Mailbox == "" {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		j := s.Store.StartSync(payload.TenantID, payload.Mailbox)
		s.Auditor.Record(audit.Event{Action: "CONNECT_SYNC", TenantID: payload.TenantID, Mailbox: payload.Mailbox})
		writeJSON(w, j)
		return
	}
	if r.Method == http.MethodGet {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		j, ok := s.Store.GetJob(id)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, j)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) connectAutodiscover(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	xml, err := autodiscover.Render(email, "/api/v1/connect/sync")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(xml))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
