// Package api — HTTP handlers ERA Mail Server (Go layer, ADR-0027).
package api

import (
	"encoding/json"
	"net/http"

	"era/services/comms/calendar/caldav"
	"era/services/comms/calendar/store"
	"era/services/comms/mail/internal/activesync"
	"era/services/comms/mail/internal/audit"
	"era/services/comms/mail/internal/auditapi"
	"era/services/comms/mail/internal/autodiscover"
	"era/services/comms/mail/internal/calendaraudit"
	"era/services/comms/mail/internal/carddav"
	"era/services/comms/mail/internal/coreclient"
	"era/services/comms/mail/internal/ews"
	"era/services/comms/mail/internal/internalapi"
	"era/services/comms/mail/internal/policy"
	"era/services/comms/mail/internal/repo"
	"era/services/comms/internal/httpauth"
	"era/services/platform/licensegate"
	"era/services/platform/tenant"
)

const serviceName = "era-mail-server"

// Config — зависимости HTTP-сервера mail-api.
type Config struct {
	Tenants   *tenant.Store
	Policies  *policy.Store
	Audit     *audit.Writer
	Core      *coreclient.Client
	Gate      *licensegate.Gate
	Repo      repo.Backend
	CalStore  store.Backend
	HTTPPort  int
	MailHost  string
	UseTLS    bool
}

// Server регистрирует маршруты ERA Mail Server.
type Server struct {
	cfg Config
}

// NewServer создаёт HTTP handler bundle.
func NewServer(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// Register монтирует маршруты на mux.
func (s *Server) Register(mux *http.ServeMux) {
	apiAuth := httpauth.FromEnv("ERA_MAIL_DEV", "")
	intAuth := httpauth.FromEnv("ERA_MAIL_DEV", "")

	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/readyz", s.readyz)
	mux.HandleFunc("/api/v1/status", s.status)
	mux.HandleFunc("/api/v1/policy", apiAuth.Wrap(s.getPolicy))
	mux.HandleFunc("/autodiscover/autodiscover.xml", s.autodiscover)
	mux.Handle("/internal/v1/audit", intAuth.RequireInternalHandler(&auditapi.Handler{Writer: s.cfg.Audit}))

	if s.cfg.Repo != nil {
		ih := &internalapi.Handler{Repo: s.cfg.Repo, Audit: s.cfg.Audit}
		mux.HandleFunc("/internal/v1/mail/deliver", intAuth.RequireInternal(ih.DeliverHTTP))
		mux.HandleFunc("/internal/v1/mail/list", intAuth.RequireInternal(ih.ListHTTP))
		mux.HandleFunc("/internal/v1/auth/verify", intAuth.RequireInternal(ih.VerifyHTTP))
		mux.HandleFunc("/internal/v1/mail/policy", intAuth.RequireInternal(ih.PolicyHTTP))
		s.registerMailAPI(mux, apiAuth)
	}

	if s.cfg.CalStore != nil {
		calH := &caldav.Handler{
			Store:   s.cfg.CalStore,
			Auditor: &calendaraudit.Recorder{Writer: s.cfg.Audit},
		}
		mux.Handle("/caldav/", apiAuth.WrapHandler(calH))
		mux.Handle("/.well-known/caldav", apiAuth.WrapHandler(caldav.WellKnown(s.cfg.CalStore)))
	}
	if s.cfg.Repo != nil && s.cfg.CalStore != nil {
		mux.Handle("/ews/Exchange.asmx", apiAuth.WrapHandler(&ews.Handler{
			Repo: s.cfg.Repo,
			Cal:  s.cfg.CalStore,
		}))
	}
	if s.cfg.Repo != nil {
		cardH := &carddav.Handler{Repo: s.cfg.Repo}
		mux.Handle("/carddav/", apiAuth.WrapHandler(cardH))
		mux.Handle("/.well-known/carddav", apiAuth.WrapHandler(http.HandlerFunc(carddav.WellKnown)))
		asH := &activesync.Handler{Repo: s.cfg.Repo}
		mux.Handle("/Microsoft-Server-ActiveSync", apiAuth.WrapHandler(asH))
	}
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{
		"status":  "ok",
		"service": serviceName,
		"phase":   "r-wave",
	})
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	coreStatus := s.cfg.Core.Status()
	repoMode := "memory"
	if s.cfg.Repo != nil {
		switch s.cfg.Repo.(type) {
		case *repo.Postgres:
			repoMode = "postgres"
		case *repo.Repository:
			if r, ok := s.cfg.Repo.(*repo.Repository); ok {
				if _, ok := r.Backend.(*repo.Postgres); ok {
					repoMode = "postgres"
				}
			}
		}
	}
	auditMode := "noop"
	if s.cfg.Audit != nil && s.cfg.Audit.IsConfigured() {
		auditMode = "clickhouse"
	}
	writeJSON(w, map[string]any{
		"product":  "era-communications",
		"edition":  "era-mail-server",
		"status":   "r-1",
		"licensed": s.cfg.Gate.Allow(licensegate.ModuleCommsMailServer),
		"core":     coreStatus,
		"storage":  repoMode,
		"audit":    auditMode,
		"protocols": map[string]string{
			"smtp":       "auth-tls",
			"imap":       "auth-tls",
			"caldav":     calStatus(s.cfg.CalStore),
			"carddav":    carddavStatus(s.cfg.Repo),
			"ews":        ewsStatus(s.cfg.Repo),
			"activesync": "subset",
		},
	})
}

func calStatus(st store.Backend) string {
	if st != nil {
		return "pilot"
	}
	return "planned"
}

func ewsStatus(r repo.Backend) string {
	if r != nil {
		return "v2-subset"
	}
	return "planned"
}

func carddavStatus(r repo.Backend) string {
	if r != nil {
		return "pilot"
	}
	return "planned"
}

func (s *Server) getPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}
	if s.cfg.Repo != nil {
		if p, ok := s.cfg.Repo.GetPolicy(tenantID); ok {
			writeJSON(w, policy.InlinePolicy{
				MaxAttachmentSizeMB:      p.MaxAttachmentSizeMB,
				QuotaMBPerUser:           p.QuotaMBPerUser,
				RetentionDays:            p.RetentionDays,
				MaxAttachmentsPerMessage: p.MaxAttachmentsPerMessage,
			})
			return
		}
	}
	p, ok := s.cfg.Policies.Get(tenantID)
	if !ok {
		p = policy.DefaultPolicy()
	}
	writeJSON(w, p)
}

func (s *Server) autodiscover(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email query param required", http.StatusBadRequest)
		return
	}
	host := envOr(r, "ERA_MAIL_HOST", "mail.mail.gov.az")
	useTLS := s.cfg.UseTLS
	smtpTLS := useTLS
	xml, err := autodiscover.Render(autodiscover.Config{
		Email:      email,
		Tenants:    s.cfg.Tenants,
		MailHost:   host,
		IMAPHost:   host,
		SMTPHost:   host,
		EWSHost:    host,
		CalDAVHost: host,
		IMAPPort:   993,
		SMTPPort:   587,
		HTTPPort:   s.cfg.HTTPPort,
		SMTPUseTLS: smtpTLS,
		UseTLS:     useTLS,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(xml))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func envOr(r *http.Request, key, def string) string {
	if v := r.Header.Get("X-ERA-Mail-Host"); v != "" {
		return v
	}
	return def
}
