package api

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"era/services/cloud-portal/internal/rbac"
	"era/services/cloud-portal/internal/store"
	lic "era/services/license/pkg/license"
	"era/services/platform/metrics"
	"era/services/platform/tlsutil"
	"github.com/google/uuid"
)

// Server — ERA Cloud Portal v0 (ADR-0018 §7).
type Server struct {
	Store *store.Store
	Priv  ed25519.PrivateKey
	Pub   ed25519.PublicKey
}

func New(st *store.Store, priv ed25519.PrivateKey, pub ed25519.PublicKey) *Server {
	return &Server{Store: st, Priv: priv, Pub: pub}
}

func LoadKeys() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	raw := os.Getenv("ERA_VENDOR_PRIV")
	if raw == "" {
		if p := os.Getenv("ERA_VENDOR_PRIV_FILE"); p != "" {
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, nil, err
			}
			raw = strings.TrimSpace(string(b))
		}
	}
	if raw == "" {
		pub, priv, err := lic.GenerateKeypair()
		return priv, pub, err
	}
	priv, err := lic.DecodePrivateKey(raw)
	if err != nil {
		return nil, nil, err
	}
	return priv, priv.Public().(ed25519.PublicKey), nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/api/v1/installations", s.handleInstallations)
	mux.HandleFunc("/api/v1/hybrid/lease/renew", s.handleLeaseRenew)
	mux.HandleFunc("/api/v1/hybrid/crl", s.handleCRL)
	mux.HandleFunc("/api/v1/hybrid/health", s.handleHealth)
	mux.HandleFunc("/api/v1/managed/", s.handleManaged)
	return rbac.Middleware(mux)
}

func (s *Server) handleInstallations(w http.ResponseWriter, r *http.Request) {
	if !rbac.CanReadInstallations(rbac.FromRequest(r)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"installations": s.Store.ListInstallations()})
	case http.MethodPost:
		if rbac.FromRequest(r) != rbac.RoleVendorAdmin {
			http.Error(w, "admin only", http.StatusForbidden)
			return
		}
		var inst store.Installation
		if err := json.NewDecoder(r.Body).Decode(&inst); err != nil || inst.DeploymentID == "" {
			http.Error(w, "deployment_id required", http.StatusBadRequest)
			return
		}
		s.Store.UpsertInstallation(&inst)
		writeJSON(w, http.StatusCreated, inst)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLeaseRenew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deployment := r.URL.Query().Get("deployment_id")
	licenseID := r.URL.Query().Get("license_id")
	if deployment == "" || licenseID == "" {
		http.Error(w, "deployment_id and license_id required", http.StatusBadRequest)
		return
	}
	inst, ok := s.Store.GetInstallation(deployment)
	if !ok {
		inst = &store.Installation{DeploymentID: deployment, LicenseID: licenseID, TenantID: "unknown"}
		s.Store.UpsertInstallation(inst)
	}
	def := lic.DefaultLeasePolicy()
	now := time.Now().UTC()
	claims := &lic.LeaseClaims{
		LicenseID:            licenseID,
		DeploymentID:         deployment,
		TenantID:             inst.TenantID,
		IssuedAt:             now.Unix(),
		ExpiresAt:            now.AddDate(0, 0, 30).Unix(),
		GraceDays:            def.GraceDays,
		OfflineMaxDays:       def.OfflineMaxDays,
		RenewalIntervalHours: def.RenewalIntervalHours,
		DegradationMode:      def.DegradationMode,
	}
	token, err := lic.SignLease(claims, s.Priv)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s *Server) handleCRL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := s.Store.CRL()
	if token == "" {
		crl := &lic.CRL{IssuedAt: time.Now().UTC().Unix(), Revoked: []string{}}
		var err error
		token, err = lic.SignCRL(crl, s.Priv)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.Store.SetCRL(token)
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	raw, _ := json.Marshal(payload)
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"cmdline", "raw_event", "pii", "case_body", "lake"} {
		if strings.Contains(lower, forbidden) {
			http.Error(w, "forbidden payload field", http.StatusBadRequest)
			return
		}
	}
	deployment, _ := payload["deployment_id"].(string)
	if deployment == "" {
		http.Error(w, "deployment_id required", http.StatusBadRequest)
		return
	}
	s.Store.RecordHealth(deployment, payload)
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted", "id": uuid.NewString()})
}

func (s *Server) handleManaged(w http.ResponseWriter, r *http.Request) {
	if rbac.ForbiddenRawData(r.URL.Path) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !rbac.CanReadInstallations(rbac.FromRequest(r)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"view":          "managed",
		"installations": s.Store.ListInstallations(),
		"scopes":        []string{"installations", "health", "licenses", "versions"},
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func ListenAddr(addr string, handler http.Handler) error {
	tlsCfg := tlsutil.ServerFromEnv()
	srv := tlsCfg.HTTPServer(addr, handler)
	return tlsCfg.Listen(srv)
}
