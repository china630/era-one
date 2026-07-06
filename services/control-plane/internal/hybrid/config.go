// Package hybrid — ERA Hybrid Relay (модуль control-plane, ADR-0018 §3).
package hybrid

import (
	"os"
	"strings"

	"era/services/control-plane/internal/store"
)

// Config — runtime-конфиг Relay (env + tenant policy).
type Config struct {
	Enabled      bool
	PortalURL    string
	UpdateURL    string
	Allowlist    []string
	HealthLevel  string
	VendorPubKey string
	DeploymentID string
	LicenseID    string
}

// LoadConfig читает env и накладывает tenant policy из store.
func LoadConfig(st store.Repository) Config {
	pol := st.GetHybridPolicy()
	cfg := Config{
		Enabled:     pol.Enabled,
		PortalURL:   pol.PortalURL,
		UpdateURL:   pol.UpdateURL,
		Allowlist:   pol.EgressAllowlist,
		HealthLevel: pol.HealthLevel,
		DeploymentID: pol.DeploymentID,
		LicenseID:    pol.LicenseID,
	}
	if v := os.Getenv("ERA_HYBRID_CONNECTED"); v == "1" || strings.EqualFold(v, "true") {
		cfg.Enabled = true
	}
	if u := os.Getenv("ERA_PORTAL_URL"); u != "" {
		cfg.PortalURL = u
	}
	if u := os.Getenv("ERA_UPDATE_URL"); u != "" {
		cfg.UpdateURL = u
	}
	if pub := os.Getenv("ERA_VENDOR_PUB"); pub != "" {
		cfg.VendorPubKey = pub
	}
	if cfg.HealthLevel == "" {
		cfg.HealthLevel = "A"
	}
	if len(cfg.Allowlist) == 0 {
		cfg.Allowlist = defaultAllowlist(cfg.PortalURL, cfg.UpdateURL)
	}
	return cfg
}

func defaultAllowlist(portal, update string) []string {
	var out []string
	for _, u := range []string{portal, update} {
		if h := hostFromURL(u); h != "" {
			out = append(out, h)
		}
	}
	return out
}

func hostFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	if i := strings.Index(raw, "/"); i >= 0 {
		raw = raw[:i]
	}
	if i := strings.Index(raw, ":"); i >= 0 {
		raw = raw[:i]
	}
	return raw
}

// HostAllowed проверяет egress allowlist.
func HostAllowed(allowlist []string, host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	for _, a := range allowlist {
		if strings.EqualFold(strings.TrimSpace(a), host) {
			return true
		}
	}
	return false
}
