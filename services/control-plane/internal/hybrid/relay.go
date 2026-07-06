package hybrid

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"era/services/control-plane/internal/store"
	lic "era/services/license/pkg/license"
)

// Relay — outbound worker Hybrid Relay (ADR-0018 §3).
type Relay struct {
	Store  store.Repository
	Config Config
	Doer   *HTTPDoer
	Pub    ed25519.PublicKey
}

// NewRelay создаёт Relay из store и env.
func NewRelay(st store.Repository) (*Relay, error) {
	cfg := LoadConfig(st)
	r := &Relay{Store: st, Config: cfg}
	if cfg.VendorPubKey != "" {
		pub, err := lic.DecodePublicKey(strings.TrimSpace(cfg.VendorPubKey))
		if err != nil {
			return nil, err
		}
		r.Pub = pub
	}
	r.Doer = NewHTTPDoer(cfg.Allowlist, st, cfg.HealthLevel)
	return r, nil
}

// Start запускает фоновый цикл синхронизации, если connected включён.
func (r *Relay) Start(ctx context.Context) {
	if !r.Config.Enabled {
		log.Printf("hybrid relay: connected OFF (air-gap)")
		return
	}
	interval := time.Duration(envInt("ERA_HYBRID_SYNC_SEC", 60)) * time.Second
	log.Printf("hybrid relay: connected ON, sync every %s", interval)
	go func() {
		r.SyncOnce()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				r.SyncOnce()
			}
		}
	}()
}

// SyncOnce выполняет один цикл: lease, CRL, bundle, health A.
func (r *Relay) SyncOnce() {
	cfg := LoadConfig(r.Store)
	r.Config = cfg
	rt := r.Store.GetHybridRuntime()
	var lastErr string

	if cfg.PortalURL != "" && r.Pub != nil {
		if err := r.pullLease(cfg, &rt); err != nil {
			lastErr = err.Error()
		}
		if err := r.pullCRL(cfg); err != nil && lastErr == "" {
			lastErr = err.Error()
		}
		if cfg.HealthLevel == "A" {
			if err := r.pushHealthA(cfg, rt); err != nil && lastErr == "" {
				lastErr = err.Error()
			}
		}
		if cfg.HealthLevel == "B" || cfg.HealthLevel == "C" {
			if err := r.pushHealthB(cfg, rt); err != nil && lastErr == "" {
				lastErr = err.Error()
			}
		}
	}
	if cfg.UpdateURL != "" && r.Pub != nil {
		if err := r.pullBundle(cfg, &rt); err != nil && lastErr == "" {
			lastErr = err.Error()
		}
	}
	rt.LastSyncAt = time.Now().UTC()
	rt.LastError = lastErr
	r.Store.SetHybridRuntime(rt)
}

func (r *Relay) pullLease(cfg Config, rt *store.HybridRuntime) error {
	q := fmt.Sprintf("/api/v1/hybrid/lease/renew?deployment_id=%s&license_id=%s",
		urlQuery(cfg.DeploymentID), urlQuery(cfg.LicenseID))
	raw, _, err := r.Doer.Get(joinURL(cfg.PortalURL, q), "lease_renew")
	if err != nil {
		return err
	}
	var resp LeaseResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}
	claims, err := lic.VerifyLease(resp.Token, r.Pub)
	if err != nil {
		return err
	}
	if cfg.DeploymentID != "" && claims.DeploymentID != cfg.DeploymentID {
		return fmt.Errorf("lease deployment mismatch")
	}
	token, renew := r.Store.GetLeaseCache()
	_ = token
	ev := claims.EvaluateLease(time.Now().UTC(), cfg.LicenseID, cfg.DeploymentID, renew)
	rt.LeaseStatus = string(ev.Status)
	rt.LeaseMessage = ev.Message
	rt.LastLeaseRenew = time.Now().UTC()
	r.Store.SetLeaseCache(resp.Token, rt.LastLeaseRenew)
	r.Store.RecordAudit("hybrid.lease_renew", "relay", cfg.DeploymentID, string(ev.Status))
	return nil
}

func (r *Relay) pullCRL(cfg Config) error {
	raw, _, err := r.Doer.Get(joinURL(cfg.PortalURL, "/api/v1/hybrid/crl"), "crl_pull")
	if err != nil {
		return err
	}
	var resp CRLResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}
	if _, err := lic.VerifyCRL(resp.Token, r.Pub); err != nil {
		return err
	}
	rt := r.Store.GetHybridRuntime()
	rt.LastCRLIssuedAt = time.Now().UTC()
	r.Store.SetHybridRuntime(rt)
	r.Store.RecordAudit("hybrid.crl_pull", "relay", cfg.DeploymentID, "ok")
	return nil
}

func (r *Relay) pullBundle(cfg Config, rt *store.HybridRuntime) error {
	raw, _, err := r.Doer.Get(joinURL(cfg.UpdateURL, "/api/v1/bundles/latest"), "bundle_pull")
	if err != nil {
		return err
	}
	var resp BundleResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}
	bundle, err := verifyBundleToken(resp.Token, r.Pub)
	if err != nil {
		return err
	}
	applyBundleToPolicy(r.Store, bundle)
	rt.LastBundleID = bundle.BundleID
	r.Store.SetHybridRuntime(*rt)
	r.Store.RecordAudit("hybrid.bundle_apply", "relay", bundle.BundleID, bundle.Kind)
	return nil
}

func (r *Relay) pushHealthB(cfg Config, rt store.HybridRuntime) error {
	pol := r.Store.GetHybridPolicy()
	h := BuildHealthB(r.Store, pol, rt)
	body, err := json.Marshal(h)
	if err != nil {
		return err
	}
	safe := RedactHealthJSON(string(body))
	if ContainsForbiddenEgress(safe) || HealthBForbiddenFields(safe) {
		return fmt.Errorf("health B payload failed redaction gate")
	}
	var payload any
	if err := json.Unmarshal([]byte(safe), &payload); err != nil {
		return err
	}
	_, _, err = r.Doer.PostJSON(joinURL(cfg.PortalURL, "/api/v1/hybrid/health"), "health_b", payload)
	return err
}

func (r *Relay) pushHealthA(cfg Config, rt store.HybridRuntime) error {
	pol := r.Store.GetHybridPolicy()
	h := BuildHealthA(r.Store, pol, rt)
	body, err := json.Marshal(h)
	if err != nil {
		return err
	}
	safe := RedactHealthJSON(string(body))
	if ContainsForbiddenEgress(safe) {
		return fmt.Errorf("health payload failed redaction gate")
	}
	var payload any
	if err := json.Unmarshal([]byte(safe), &payload); err != nil {
		return err
	}
	_, _, err = r.Doer.PostJSON(joinURL(cfg.PortalURL, "/api/v1/hybrid/health"), "health_a", payload)
	return err
}

func urlQuery(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return def
}
