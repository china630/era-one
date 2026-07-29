package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"era/services/comms/mail/internal/repo"
)

func (s *Server) registerMailAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/mailboxes", s.mailboxes)
	mux.HandleFunc("/api/v1/mailboxes/", s.mailboxByEmail)
	mux.HandleFunc("/api/v1/mail/messages", s.mailMessages)
	mux.HandleFunc("/api/v1/mail/send", s.mailSend)
}

type createMailboxReq struct {
	TenantID   string `json:"tenant_id"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	QuotaBytes int64  `json:"quota_bytes"`
}

func (s *Server) mailboxes(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Repo == nil {
		http.Error(w, "repo unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodPost:
		var req createMailboxReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.QuotaBytes == 0 {
			req.QuotaBytes = 512 << 20
		}
		if req.TenantID == "" {
			req.TenantID = "t-demo"
		}
		mb, err := s.cfg.Repo.CreateMailbox(req.TenantID, req.Email, req.Password, req.QuotaBytes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, mb)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) mailboxByEmail(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Repo == nil {
		http.Error(w, "repo unavailable", http.StatusServiceUnavailable)
		return
	}
	email := strings.TrimPrefix(r.URL.Path, "/api/v1/mailboxes/")
	email = strings.TrimSpace(email)
	if email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		mb, err := s.cfg.Repo.GetMailboxByEmail(email)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, mb)
	case http.MethodPatch:
		var patch repo.MailboxPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		updated, err := s.cfg.Repo.UpdateMailbox(email, patch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, updated)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) mailMessages(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Repo == nil || r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}
	msgs, err := s.cfg.Repo.ListMessages(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{"messages": msgs})
}

type sendMailReq struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (s *Server) mailSend(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Repo == nil || r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req sendMailReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	raw := "From: " + req.From + "\r\nTo: " + req.To + "\r\nSubject: " + req.Subject + "\r\n\r\n" + req.Body
	if p, ok := s.cfg.Repo.GetPolicy("t-demo"); ok {
		maxBytes := int64(p.MaxAttachmentSizeMB) * 1024 * 1024
		if int64(len(raw)) > maxBytes {
			http.Error(w, "message too large", http.StatusRequestEntityTooLarge)
			return
		}
	}
	msg, err := s.cfg.Repo.DeliverRaw(req.To, []byte(raw), req.From)
	if err != nil {
		if strings.Contains(err.Error(), "quota") {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"id": msg.ID, "uid": msg.UID})
}
