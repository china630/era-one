package proxy

import (
	"io"
	"net/http"
	"strings"

	"era/services/comms/mail-bridge/internal/audit"
	"era/services/comms/mail-bridge/internal/upstream"
)

// EWS proxies SOAP to upstream backend.
type EWS struct {
	Router *upstream.Router
	Email  string
	Audit  *audit.Recorder
}

func (p *EWS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	email := p.Email
	if email == "" {
		email = r.Header.Get("X-ERA-Mailbox")
	}
	if email == "" {
		email = "user@mail.gov.az"
	}
	action := r.Header.Get("SOAPAction")
	if action == "" {
		action = detectAction(string(body))
	}
	be := p.Router.Resolve(email)
	status, resp, err := be.ProxyEWS(r.Context(), action, body, r.Header)
	if p.Audit != nil {
		p.Audit.Record("BRIDGE_EWS_"+strings.Trim(action, `"`), email)
	}
	if err != nil && status == 0 {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(resp)
}

func detectAction(raw string) string {
	switch {
	case strings.Contains(raw, "FindFolder"):
		return "FindFolder"
	case strings.Contains(raw, "SyncFolderItems"):
		return "SyncFolderItems"
	case strings.Contains(raw, "CreateItem"):
		return "CreateItem"
	default:
		return "Unknown"
	}
}

// Ensure EWS implements http.Handler
var _ http.Handler = (*EWS)(nil)
