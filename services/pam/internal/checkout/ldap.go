package checkout

import (
	"os"
	"strings"
)

// ApproverInLDAPGroups проверяет членство в ERA_LDAP_APPROVER_GROUPS (allowlist).
// groups — из LDAP/SSO (в dev — заголовок X-ERA-Groups).
func ApproverInLDAPGroups(groups []string) bool {
	raw := strings.TrimSpace(os.Getenv("ERA_LDAP_APPROVER_GROUPS"))
	if raw == "" {
		return true
	}
	allow := map[string]struct{}{}
	for _, g := range strings.Split(raw, ",") {
		g = strings.TrimSpace(g)
		if g != "" {
			allow[strings.ToLower(g)] = struct{}{}
		}
	}
	for _, g := range groups {
		if _, ok := allow[strings.ToLower(strings.TrimSpace(g))]; ok {
			return true
		}
	}
	return false
}

// ParseGroups разбирает CSV групп из SSO/LDAP claim.
func ParseGroups(csv string) []string {
	if csv == "" {
		return nil
	}
	var out []string
	for _, g := range strings.Split(csv, ",") {
		g = strings.TrimSpace(g)
		if g != "" {
			out = append(out, g)
		}
	}
	return out
}
