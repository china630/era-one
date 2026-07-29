package upstream

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Route describes upstream for an email domain or mailbox.
type Route struct {
	Type        string `json:"type"`
	BaseURL     string `json:"base_url"`
	IMAPHost    string `json:"imap_host"`
	IMAPPort    int    `json:"imap_port"`
	IMAPUser    string `json:"imap_user"`
	IMAPPassRef string `json:"imap_password_ref"`
	SMTPHost    string `json:"smtp_host"`
	SMTPPort    int    `json:"smtp_port"`
}

// Router maps email addresses to upstream backends.
type Router struct {
	routes   map[string]Route
	backends map[string]Backend
	defaultB Backend
}

func NewRouter(defaultB Backend) *Router {
	return &Router{
		routes:   make(map[string]Route),
		backends: make(map[string]Backend),
		defaultB: defaultB,
	}
}

// LoadFromEnv reads ERA_BRIDGE_UPSTREAM_JSON and ERA_BRIDGE_TENANT_MAP.
func LoadFromEnv(defaultB Backend) (*Router, error) {
	r := NewRouter(defaultB)
	for domain, be := range LoadTenantMap(defaultB) {
		r.backends[domain] = be
		r.routes[domain] = Route{Type: be.Name()}
	}
	raw := os.Getenv("ERA_BRIDGE_UPSTREAM_JSON")
	if raw == "" {
		return r, nil
	}
	var m map[string]Route
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	for domain, route := range m {
		r.routes[domain] = route
		switch strings.ToLower(route.Type) {
		case "stub", "":
			r.backends[domain] = defaultB
		default:
			r.backends[domain] = BuildBackend(route, defaultB)
		}
	}
	return r, nil
}

func (r *Router) Resolve(email string) Backend {
	email = strings.ToLower(strings.TrimSpace(email))
	if i := strings.LastIndex(email, "@"); i >= 0 {
		domain := email[i+1:]
		if b, ok := r.backends[domain]; ok {
			return b
		}
		if _, ok := r.routes[domain]; ok {
			return r.defaultB
		}
	}
	return r.defaultB
}

func (r *Router) RouteInfo(email string) (Route, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	idx := strings.LastIndex(email, "@")
	if idx < 0 {
		return Route{}, fmt.Errorf("invalid email")
	}
	domain := email[idx+1:]
	if rt, ok := r.routes[domain]; ok {
		return rt, nil
	}
	return Route{Type: "stub"}, nil
}
