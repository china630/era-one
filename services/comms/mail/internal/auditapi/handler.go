// Package auditapi — internal webhook от mail-core (CM1-7).
package auditapi

import (
	"encoding/json"
	"net/http"

	"era/services/comms/mail/internal/audit"
	erav1 "era/contracts/gen/era/v1"
)

// PostAuditRequest — тело webhook от Rust mail-core.
type PostAuditRequest struct {
	TenantID  string `json:"tenant_id"`
	Mailbox   string `json:"mailbox"`
	Action    string `json:"action"`
	MailFrom  string `json:"mail_from"`
	MessageID string `json:"message_id"`
	SrcIP     string `json:"src_ip"`
}

// Handler принимает audit events от mail-core.
type Handler struct {
	Writer *audit.Writer
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req PostAuditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	action := erav1.MailAuditAction_MAIL_AUDIT_SEND
	if req.Action == "receive" {
		action = erav1.MailAuditAction_MAIL_AUDIT_RECEIVE
	}
	ev := &erav1.MailAuditEvent{
		TenantId:  req.TenantID,
		Mailbox:   req.Mailbox,
		Action:    action,
		MessageId: req.MessageID,
		SrcIp:     req.SrcIP,
		Metadata:  map[string]string{},
	}
	if req.MailFrom != "" {
		ev.Metadata["mail_from"] = req.MailFrom
	}
	if err := h.Writer.Insert(r.Context(), ev); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
