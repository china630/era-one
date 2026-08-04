// Package rbac — RBAC with trust boundary (Scaffold-Green Wave 1).
package rbac

import (
	"net"
	"net/http"
	"os"
	"strings"
)

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleAnalyst Role = "analyst"
	RoleViewer  Role = "viewer"
	RoleAgent   Role = "agent" // service token: read-only policy fetch, never admin
	RoleNone    Role = ""
)

// TrustMode controls whether client X-ERA-* headers are trusted.
type TrustMode string

const (
	TrustDev    TrustMode = "dev"     // lab: accept client headers (legacy)
	TrustProxy  TrustMode = "proxy"   // Trusted-Proxy header AND trusted hop
	TrustAPIKey TrustMode = "api_key" // admin only via ERA_API_KEY Bearer
)

// TrustFromEnv resolves ERA_RBAC_TRUST with production defaults.
func TrustFromEnv() TrustMode {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("ERA_RBAC_TRUST")))
	switch raw {
	case "dev", "proxy", "api_key":
		return TrustMode(raw)
	}
	if envTruthy("ERA_PRODUCTION") || envTruthy("ERA_LICENSE_STRICT") || envTruthy("ERA_ENV_PRODUCTION") {
		if strings.TrimSpace(os.Getenv("ERA_API_KEY")) != "" {
			return TrustAPIKey
		}
		return TrustProxy
	}
	if strings.EqualFold(os.Getenv("ERA_ENV"), "production") {
		if strings.TrimSpace(os.Getenv("ERA_API_KEY")) != "" {
			return TrustAPIKey
		}
		return TrustProxy
	}
	return TrustDev
}

func envTruthy(k string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes"
}

func headersTrusted(r *http.Request, mode TrustMode) bool {
	switch mode {
	case TrustDev:
		return true
	case TrustProxy:
		if r.Header.Get("X-ERA-Trusted-Proxy") != "1" {
			return false
		}
		// Header alone is not enough — client can spoof it. Require trusted hop.
		return isTrustedHop(r)
	case TrustAPIKey:
		return false // roles from headers never trusted; only API key
	default:
		return false
	}
}

// isTrustedHop is true for loopback/unix peers or ERA_TRUSTED_PROXY_CIDRS.
func isTrustedHop(r *http.Request) bool {
	host := remoteHost(r)
	if host == "" {
		return false
	}
	// Unix domain sockets (Go often reports "@" or path-like RemoteAddr).
	if host == "@" || strings.HasPrefix(host, "/") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	for _, cidr := range trustedProxyCIDRs() {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func remoteHost(r *http.Request) string {
	addr := strings.TrimSpace(r.RemoteAddr)
	if addr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func trustedProxyCIDRs() []*net.IPNet {
	raw := strings.TrimSpace(os.Getenv("ERA_TRUSTED_PROXY_CIDRS"))
	if raw == "" {
		return nil
	}
	var out []*net.IPNet
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		_, n, err := net.ParseCIDR(part)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

// FromRequest resolves role under current trust mode.
func FromRequest(r *http.Request) Role {
	return FromRequestTrust(r, TrustFromEnv())
}

// FromRequestTrust is the testable core.
func FromRequestTrust(r *http.Request, mode TrustMode) Role {
	if key := os.Getenv("ERA_API_KEY"); key != "" {
		if r.Header.Get("Authorization") == "Bearer "+key {
			return RoleAdmin
		}
	}
	// Agent service token is NOT admin — read-only via IsTrustedAgent.
	if bearerAgent(r) {
		return RoleAgent
	}
	if mode == TrustAPIKey {
		// No API key match → no elevated role from headers
		return RoleNone
	}
	if !headersTrusted(r, mode) {
		return RoleNone
	}
	switch strings.ToLower(r.Header.Get("X-ERA-Role")) {
	case "admin":
		return RoleAdmin
	case "viewer":
		return RoleViewer
	case "analyst":
		return RoleAnalyst
	case "agent":
		return RoleAgent
	case "":
		if mode == TrustDev {
			return RoleAnalyst // legacy lab default
		}
		return RoleNone
	default:
		if mode == TrustDev {
			return RoleAnalyst
		}
		return RoleNone
	}
}

// Actor returns audit actor; ignores spoofed X-ERA-Actor when headers untrusted.
func Actor(r *http.Request) string {
	return ActorTrust(r, TrustFromEnv())
}

func ActorTrust(r *http.Request, mode TrustMode) string {
	if bearerAdmin(r) {
		if a := r.Header.Get("X-ERA-Actor"); a != "" {
			return a
		}
		return "api-key"
	}
	if !headersTrusted(r, mode) {
		return "anonymous"
	}
	if a := r.Header.Get("X-ERA-Actor"); a != "" {
		return a
	}
	return "unknown"
}

// bearerAdmin is true only for ERA_API_KEY (not ERA_AGENT_TOKEN).
func bearerAdmin(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	key := os.Getenv("ERA_API_KEY")
	return key != "" && auth == "Bearer "+key
}

func bearerAgent(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	tok := os.Getenv("ERA_AGENT_TOKEN")
	return tok != "" && auth == "Bearer "+tok
}

// IsTrustedAgent is true when actor era-agent is allowed under trust rules.
func IsTrustedAgent(r *http.Request) bool {
	return IsTrustedAgentTrust(r, TrustFromEnv())
}

// IsTrustedAgentTrust: Bearer ERA_AGENT_TOKEN, or trusted headers + X-ERA-Actor=era-agent.
func IsTrustedAgentTrust(r *http.Request, mode TrustMode) bool {
	if bearerAgent(r) {
		return true
	}
	if !headersTrusted(r, mode) {
		return false
	}
	return r.Header.Get("X-ERA-Actor") == "era-agent"
}

// Middleware rejects unauthenticated API calls under TrustProxy / TrustAPIKey.
// TrustDev remains lab-permissive. Healthz, metrics, and static UI stay open.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if skipAuthz(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		mode := TrustFromEnv()
		if mode == TrustDev {
			next.ServeHTTP(w, r)
			return
		}
		if FromRequestTrust(r, mode) == RoleNone && !IsTrustedAgentTrust(r, mode) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func skipAuthz(path string) bool {
	switch path {
	case "/healthz", "/metrics":
		return true
	}
	if path == "/" || path == "/ui" || path == "/ui/" || strings.HasPrefix(path, "/ui/") {
		return true
	}
	if strings.HasPrefix(path, "/control-assets/") {
		return true
	}
	return false
}

func CanWriteCases(r Role) bool {
	return r == RoleAdmin || r == RoleAnalyst
}

func CanReadCases(r Role) bool {
	return r == RoleAdmin || r == RoleAnalyst || r == RoleViewer
}

func IsAdmin(r Role) bool {
	return r == RoleAdmin
}
