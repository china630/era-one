package upstream

import (
	"os"
	"strings"

	icewarpbe "era/services/comms/mail-bridge/internal/upstream/icewarp"
	exchangebe "era/services/comms/mail-bridge/internal/upstream/exchange"
	imapgen "era/services/comms/mail-bridge/internal/upstream/imap_generic"
	"era/services/comms/mail-bridge/internal/upstream/synthetic"
)

// BuildBackend constructs a production upstream from route metadata.
func BuildBackend(route Route, defaultB Backend) Backend {
	switch strings.ToLower(route.Type) {
	case "icewarp":
		url := route.BaseURL
		if url == "" {
			url = os.Getenv("ERA_BRIDGE_UPSTREAM_ICEWARP_BASE_URL")
		}
		if url == "" {
			return defaultB
		}
		return icewarpbe.New(url)
	case "exchange", "exchange_onprem":
		url := route.BaseURL
		if url == "" {
			url = os.Getenv("ERA_BRIDGE_UPSTREAM_EXCHANGE_BASE_URL")
		}
		if url == "" {
			return defaultB
		}
		return exchangebe.New(url)
	case "imap_generic", "communigate", "cg":
		cfg := imapgen.Config{
			IMAPHost:    route.IMAPHost,
			IMAPPort:    route.IMAPPort,
			IMAPUser:    route.IMAPUser,
			IMAPPassRef: route.IMAPPassRef,
			SMTPHost:    route.SMTPHost,
			SMTPPort:    route.SMTPPort,
		}
		if cfg.IMAPHost == "" {
			cfg = imapgen.ConfigFromEnv()
		}
		if cfg.IMAPHost == "" {
			return defaultB
		}
		return imapgen.New(cfg)
	case "synthetic":
		return synthetic.Backend{}
	default:
		return defaultB
	}
}

// LoadTenantMap reads ERA_BRIDGE_TENANT_MAP domain:type,base_url pairs.
// Example: mail.lab.local:icewarp,cg.lab.local:imap_generic
func LoadTenantMap(defaultB Backend) map[string]Backend {
	out := make(map[string]Backend)
	raw := os.Getenv("ERA_BRIDGE_TENANT_MAP")
	if raw == "" {
		return out
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		domain := strings.ToLower(strings.TrimSpace(kv[0]))
		typ := strings.TrimSpace(kv[1])
		route := Route{Type: typ}
		if typ == "icewarp" {
			route.BaseURL = os.Getenv("ERA_BRIDGE_UPSTREAM_ICEWARP_BASE_URL")
		}
		out[domain] = BuildBackend(route, defaultB)
	}
	return out
}
