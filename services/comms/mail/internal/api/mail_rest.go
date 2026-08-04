package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"era/services/comms/internal/httpauth"
	"era/services/comms/mail/internal/repo"
)

func (s *Server) registerMailAPI(mux *http.ServeMux, auth httpauth.Config) {
	mux.HandleFunc("/api/v1/mailboxes", auth.Wrap(s.mailboxes))
	mux.HandleFunc("/api/v1/mailboxes/", auth.Wrap(s.mailboxByEmail))
	mux.HandleFunc("/api/v1/mail/messages", auth.Wrap(s.mailMessages))
	mux.HandleFunc("/api/v1/mail/send", auth.Wrap(s.mailSend))
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
	From    string          `json:"from"`
	To      json.RawMessage `json:"to"`
	Subject string          `json:"subject"`
	Body    string          `json:"body"`
}

func parseRecipients(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		parts := strings.Split(one, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err == nil {
		return many
	}
	return nil
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
	recipients := parseRecipients(req.To)
	if len(recipients) == 0 {
		http.Error(w, "to required", http.StatusBadRequest)
		return
	}
	tenantID := "t-demo"
	if mb, err := s.cfg.Repo.GetMailboxByEmail(req.From); err == nil && mb != nil && mb.TenantID != "" {
		tenantID = mb.TenantID
	}
	if p, ok := s.cfg.Repo.GetPolicy(tenantID); ok {
		maxBytes := int64(p.MaxAttachmentSizeMB) * 1024 * 1024
		rawProbe := "From: " + req.From + "\r\nTo: " + strings.Join(recipients, ",") + "\r\nSubject: " + req.Subject + "\r\n\r\n" + req.Body
		if int64(len(rawProbe)) > maxBytes {
			http.Error(w, "message too large", http.StatusRequestEntityTooLarge)
			return
		}
		if p.MaxRecipients > 0 && len(recipients) > p.MaxRecipients {
			http.Error(w, "too many recipients", http.StatusBadRequest)
			return
		}
		for _, d := range p.AttachmentExtDeny {
			if d != "" && strings.Contains(strings.ToLower(req.Subject+" "+req.Body), "."+strings.ToLower(d)) {
				http.Error(w, "attachment type denied", http.StatusBadRequest)
				return
			}
		}
	}
	var lastID, lastUID int64
	for _, to := range recipients {
		raw := "From: " + req.From + "\r\nTo: " + to + "\r\nSubject: " + req.Subject + "\r\n\r\n" + req.Body
		msg, err := s.cfg.Repo.DeliverRaw(to, []byte(raw), req.From)
		if err != nil {
			if strings.Contains(err.Error(), "quota") {
				http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastID, lastUID = msg.ID, msg.UID
		if s.cfg.Audit != nil {
			_ = s.cfg.Audit.RecordSend(context.Background(), tenantID, req.From, fmt.Sprintf("msg-%d", msg.ID), "127.0.0.1")
		}
	}
	writeJSON(w, map[string]any{"id": lastID, "uid": lastUID, "recipients": len(recipients)})
}
