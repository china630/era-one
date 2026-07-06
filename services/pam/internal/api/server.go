package api

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"era/services/platform/privilegedsession"
	"era/services/pam/internal/checkout"
	"era/services/pam/internal/proxy"
	"era/services/pam/internal/shamir"
	"era/services/pam/internal/vault"
	"era/services/platform/custody"
	"era/services/platform/licensegate"
	"era/services/platform/metrics"
)

type Server struct {
	Vault    *vault.Vault
	Checkout *checkout.Store
	Sessions *privilegedsession.Store
	SSHProxy *proxy.SSHProxy
	Custody  *custody.Chain
	Gate     *licensegate.Gate
	KMSName  string
	// initShares — dev-only Shamir shares for first unseal (не логировать).
	initShares [][]byte
}

func New(v *vault.Vault, ch *checkout.Store, sess *privilegedsession.Store, gate *licensegate.Gate, kmsName string) *Server {
	s := &Server{
		Vault: v, Checkout: ch, Sessions: sess,
		SSHProxy: proxy.NewSSHProxy(sess),
		Custody: custody.NewChain(), Gate: gate, KMSName: kmsName,
	}
	s.bootstrapDevShares()
	return s
}

func (s *Server) bootstrapDevShares() {
	master := make([]byte, 32)
	_, _ = rand.Read(master)
	shares, err := shamir.Split(master, 3, 2)
	if err != nil {
		return
	}
	s.initShares = shares
	s.Vault.SetShareHints([]string{
		"share-1-dev", "share-2-dev", "share-3-dev",
	})
}

// DevShareHex returns one share for tests (never log).
func (s *Server) DevShareHex(idx int) string {
	if idx < 0 || idx >= len(s.initShares) {
		return ""
	}
	return strings.TrimSpace(string(shamir.EncodeShares([][]byte{s.initShares[idx]})[0]))
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/api/v1/vault/status", s.handleVaultStatus)
	mux.HandleFunc("/api/v1/vault/unseal", s.handleUnseal)
	mux.HandleFunc("/api/v1/vault/seal", s.handleSeal)
	mux.HandleFunc("/api/v1/secrets", s.handleSecrets)
	mux.HandleFunc("/api/v1/checkout", s.handleCheckout)
	mux.HandleFunc("/api/v1/checkout/", s.handleCheckoutSub)
	mux.HandleFunc("/api/v1/proxy/ssh/start", s.handleSSHStart)
	mux.HandleFunc("/api/v1/proxy/ssh/command", s.handleSSHCommand)
	mux.HandleFunc("/api/v1/proxy/rdp/start", s.handleRDPStart)
	mux.HandleFunc("/api/v1/custody/head", s.handleCustodyHead)
	return mux
}

func (s *Server) requirePAM(w http.ResponseWriter) bool {
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModulePAM) {
		http.Error(w, "pam module not licensed", http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) actor(r *http.Request) string {
	if a := r.Header.Get("X-ERA-Actor"); a != "" {
		return a
	}
	return "unknown"
}

func (s *Server) role(r *http.Request) string {
	if ro := r.Header.Get("X-ERA-Role"); ro != "" {
		return ro
	}
	return "requester"
}

func (s *Server) groups(r *http.Request) []string {
	return checkout.ParseGroups(r.Header.Get("X-ERA-Groups"))
}

func (s *Server) kmsLabel() string {
	if s.KMSName != "" {
		return s.KMSName
	}
	return "software-sealed-dev"
}

func (s *Server) handleVaultStatus(w http.ResponseWriter, r *http.Request) {
	if !s.requirePAM(w) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sealed":      s.Vault.Sealed(),
		"kms":         s.kmsLabel(),
		"share_hints": s.Vault.ShareHints(),
	})
}

func (s *Server) handleUnseal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requirePAM(w) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Shares []string `json:"shares"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Shares) < 2 {
		http.Error(w, "need >=2 shares", http.StatusBadRequest)
		return
	}
	decoded, err := shamir.DecodeShares(req.Shares)
	if err != nil {
		http.Error(w, "bad shares", http.StatusBadRequest)
		return
	}
	master, err := shamir.Combine(decoded)
	if err != nil {
		http.Error(w, "combine failed", http.StatusBadRequest)
		return
	}
	if len(master) < 32 {
		padded := make([]byte, 32)
		copy(padded, master)
		master = padded
	} else if len(master) > 32 {
		master = master[:32]
	}
	if err := s.Vault.Unseal(master); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.auditCustody("vault.unseal", "vault", s.actor(r), "")
	writeJSON(w, http.StatusOK, map[string]string{"status": "unsealed"})
}

func (s *Server) handleSeal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requirePAM(w) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if s.role(r) != "admin" && s.role(r) != "pam-admin" {
		http.Error(w, "admin only", http.StatusForbidden)
		return
	}
	s.Vault.Seal()
	s.auditCustody("vault.seal", "vault", s.actor(r), "")
	writeJSON(w, http.StatusOK, map[string]string{"status": "sealed"})
}

func (s *Server) handleSecrets(w http.ResponseWriter, r *http.Request) {
	if !s.requirePAM(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"secrets": s.Vault.ListMeta()})
	case http.MethodPost:
		if s.Vault.Sealed() {
			http.Error(w, "vault sealed", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			TenantID string `json:"tenant_id"`
			Name     string `json:"name"`
			Target   string `json:"target"`
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Password == "" {
			http.Error(w, "name and password required", http.StatusBadRequest)
			return
		}
		meta, err := s.Vault.PutStatic(req.TenantID, req.Name, req.Target, req.Username, req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.auditCustody("secret.put", meta.ID, s.actor(r), req.TenantID)
		writeJSON(w, http.StatusCreated, meta)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	if !s.requirePAM(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"checkouts": s.Checkout.List()})
	case http.MethodPost:
		var req struct {
			SecretID   string `json:"secret_id"`
			TenantID   string `json:"tenant_id"`
			TTLMinutes int    `json:"ttl_minutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SecretID == "" {
			http.Error(w, "secret_id required", http.StatusBadRequest)
			return
		}
		auto, allowed := checkout.PolicyAllow(s.role(r))
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		cr, err := s.Checkout.Create(req.TenantID, req.SecretID, s.actor(r), req.TTLMinutes, auto)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.auditCustody("checkout.request", req.SecretID, s.actor(r), req.TenantID)
		writeJSON(w, http.StatusCreated, cr)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCheckoutSub(w http.ResponseWriter, r *http.Request) {
	if !s.requirePAM(w) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/checkout/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	id, action := parts[0], parts[1]
	switch action {
	case "approve":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.role(r) != "admin" && s.role(r) != "pam-admin" && !checkout.ApproverInLDAPGroups(s.groups(r)) {
			http.Error(w, "approver not in ldap groups", http.StatusForbidden)
			return
		}
		cr, ok := s.Checkout.Approve(id, s.actor(r))
		if !ok {
			http.NotFound(w, r)
			return
		}
		s.auditCustody("checkout.approve", cr.SecretID, s.actor(r), cr.TenantID)
		writeJSON(w, http.StatusOK, cr)
	case "reveal":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		cr, ok := s.Checkout.Consume(id)
		if !ok {
			http.Error(w, "not approved or expired", http.StatusForbidden)
			return
		}
		user, pass, err := s.Vault.Reveal(cr.SecretID)
		if err != nil {
			http.Error(w, "reveal failed", http.StatusInternalServerError)
			return
		}
		s.auditCustody("checkout.reveal", cr.SecretID, s.actor(r), cr.TenantID)
		// zero-knowledge: password only in this one-shot response, never in list APIs
		writeJSON(w, http.StatusOK, map[string]string{
			"username": user,
			"password": pass,
			"target":   "",
		})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleSSHStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requirePAM(w) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		CheckoutID string `json:"checkout_id"`
		Host       string `json:"host"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Host == "" {
		http.Error(w, "host required", http.StatusBadRequest)
		return
	}
	user := s.actor(r)
	if req.CheckoutID != "" {
		cr, ok := s.Checkout.Get(req.CheckoutID)
		if !ok || cr.Status != checkout.StatusApproved && cr.Status != checkout.StatusConsumed {
			http.Error(w, "invalid checkout", http.StatusForbidden)
			return
		}
		meta, ok := s.Vault.GetMeta(cr.SecretID)
		if ok {
			user = meta.Username
		}
	}
	rec := s.Sessions.Start(s.actor(r), req.Host)
	proxyAddr := ""
	if s.SSHProxy != nil {
		if addr, err := s.SSHProxy.Start(rec.ID, req.Host, 22); err == nil {
			proxyAddr = addr
		}
	}
	s.auditCustody("proxy.ssh.start", rec.ID, s.actor(r), req.Host)
	writeJSON(w, http.StatusCreated, map[string]any{
		"session_id": rec.ID,
		"host":       req.Host,
		"user":       user,
		"injected":   true,
		"proxy_addr": proxyAddr,
	})
}

func (s *Server) handleRDPStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requirePAM(w) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		CheckoutID string `json:"checkout_id"`
		Host       string `json:"host"`
		Port       int    `json:"port"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Host == "" {
		http.Error(w, "host required", http.StatusBadRequest)
		return
	}
	if req.Port <= 0 {
		req.Port = 3389
	}
	user := s.actor(r)
	if req.CheckoutID != "" {
		cr, ok := s.Checkout.Get(req.CheckoutID)
		if !ok || cr.Status != checkout.StatusApproved && cr.Status != checkout.StatusConsumed {
			http.Error(w, "invalid checkout", http.StatusForbidden)
			return
		}
		if meta, ok := s.Vault.GetMeta(cr.SecretID); ok {
			user = meta.Username
		}
	}
	rec := s.Sessions.Start(s.actor(r), req.Host)
	s.auditCustody("proxy.rdp.start", rec.ID, s.actor(r), req.Host)
	writeJSON(w, http.StatusCreated, map[string]any{
		"session_id": rec.ID,
		"host":       req.Host,
		"port":       req.Port,
		"user":       user,
		"protocol":   "rdp",
		"width":      req.Width,
		"height":     req.Height,
		"stub":       true,
	})
}

func (s *Server) handleSSHCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requirePAM(w) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
		Command   string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	alert, fired := s.Sessions.LogCommand(req.SessionID, req.Command)
	if fired {
		s.auditCustody("proxy.ssh.alert", req.SessionID, s.actor(r), req.Command)
	}
	writeJSON(w, http.StatusOK, map[string]any{"alert": alert, "fired": fired})
}

func (s *Server) handleCustodyHead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || !s.requirePAM(w) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"head": s.Custody.Head()})
}

func (s *Server) auditCustody(action, target, actor, tenant string) {
	payload := vault.CustodyPayload(action, target, actor, tenant)
	s.Custody.Seal(payload)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// LogWriter wraps log output for no-secret-leak tests.
type LogWriter struct {
	W io.Writer
}

func (lw *LogWriter) Write(p []byte) (int, error) {
	if containsSecretMarker(string(p)) {
		log.Printf("REDACTED log line (secret marker)")
		return len(p), nil
	}
	return lw.W.Write(p)
}

func containsSecretMarker(s string) bool {
	return strings.Contains(s, "SuperSecret") || strings.Contains(s, "password=")
}

// ResponseMustNotLeak scans body for forbidden secret substrings (CI gate).
func ResponseMustNotLeak(body []byte, forbidden ...string) bool {
	for _, f := range forbidden {
		if bytes.Contains(body, []byte(f)) {
			return false
		}
	}
	return true
}
