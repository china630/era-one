package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"era/services/comms/internal/httpauth"
	"era/services/comms/mail-bridge/internal/audit"
	bridgead "era/services/comms/mail-bridge/internal/autodiscover"
	"era/services/comms/mail-bridge/internal/proxy"
	"era/services/comms/mail-bridge/internal/upstream"
	"era/services/platform/licensegate"
	"era/services/platform/tenant"
)

const serviceName = "era-mail-bridge"

type Server struct {
	Gate   *licensegate.Gate
	Router *upstream.Router
	Audit  *audit.Recorder
	Tenant *tenant.Store
	Port   int
	UseTLS bool
}

func NewServer(gate *licensegate.Gate, router *upstream.Router, aud *audit.Recorder) *Server {
	ts := tenant.NewStore()
	_ = ts.PutTenant(tenant.Tenant{ID: "t-demo", Name: "Demo", Slug: "demo"})
	_ = ts.PutDomain(tenant.Domain{ID: "d-mail", TenantID: "t-demo", FQDN: "mail.gov.az", Primary: true})
	port := 8151
	if v := os.Getenv("ERA_BRIDGE_HTTP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}
	return &Server{
		Gate:   gate,
		Router: router,
		Audit:  aud,
		Tenant: ts,
		Port:   port,
		UseTLS: os.Getenv("ERA_BRIDGE_TLS") == "1",
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	auth := httpauth.FromEnv("ERA_BRIDGE_DEV", "")
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/autodiscover/autodiscover.xml", s.autodiscover)
	mux.Handle("/ews/Exchange.asmx", auth.WrapHandler(&proxy.EWS{Router: s.Router, Audit: s.Audit}))
	cd := proxy.CalDAVFromEnv()
	cd.Audit = s.Audit
	mux.Handle("/caldav/", auth.WrapHandler(cd))
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModuleCommsOutlookBridge) {
		http.Error(w, "license", http.StatusForbidden)
		return
	}
	mode := "stub"
	if os.Getenv("ERA_BRIDGE_SYNTHETIC") == "1" {
		mode = "synthetic"
	} else if u := os.Getenv("ERA_BRIDGE_UPSTREAM"); u != "" {
		mode = strings.ToLower(u)
	}
	writeJSON(w, map[string]string{
		"status":         "ok",
		"service":        serviceName,
		"upstream_mode":  mode,
	})
}

func (s *Server) autodiscover(w http.ResponseWriter, r *http.Request) {
	if !s.licensed(w) {
		return
	}
	email := r.URL.Query().Get("email")
	xml, err := bridgead.Render(bridgead.Config{
		Email:      email,
		BridgeHost: r.Host,
		HTTPPort:   s.Port,
		UseTLS:     s.UseTLS,
		Tenants:    s.Tenant,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.Audit.Record("BRIDGE_AUTODISCOVER", email)
	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(xml))
}

func (s *Server) licensed(w http.ResponseWriter) bool {
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModuleCommsOutlookBridge) {
		http.Error(w, "license", http.StatusForbidden)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
