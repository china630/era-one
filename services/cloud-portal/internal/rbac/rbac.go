// Package rbac — Managed View scopes (ADR-0018 §7).
package rbac

import (
	"net/http"
	"os"
	"strings"
)

type Role string

const (
	RoleVendorAdmin Role = "vendor_admin"
	RolePartner     Role = "partner"
	RoleReadonly    Role = "readonly"
)

func FromRequest(r *http.Request) Role {
	if key := os.Getenv("ERA_API_KEY"); key != "" {
		if r.Header.Get("Authorization") == "Bearer "+key {
			return RoleVendorAdmin
		}
	}
	switch strings.ToLower(r.Header.Get("X-ERA-Role")) {
	case "vendor_admin", "admin":
		return RoleVendorAdmin
	case "partner":
		return RolePartner
	default:
		return RoleReadonly
	}
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CanReadInstallations — Managed View scope.
func CanReadInstallations(r Role) bool {
	return r == RoleVendorAdmin || r == RolePartner || r == RoleReadonly
}

// CanIssueLease — только vendor admin.
func CanIssueLease(r Role) bool {
	return r == RoleVendorAdmin
}

// ForbiddenRawData — жёсткий запрет сырья/кейсов/lake/PII.
func ForbiddenRawData(path string) bool {
	lower := strings.ToLower(path)
	for _, f := range []string{"/raw", "/cases", "/lake", "/events", "/pii"} {
		if strings.Contains(lower, f) {
			return true
		}
	}
	return false
}
