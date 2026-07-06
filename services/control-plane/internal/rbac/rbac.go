// Package rbac — skeleton RBAC (GA-1 S5-23, GA-2 SSO prep).
package rbac

import (
	"net/http"
	"os"
	"strings"
)

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleAnalyst Role = "analyst"
	RoleViewer  Role = "viewer"
)

func FromRequest(r *http.Request) Role {
	if key := os.Getenv("ERA_API_KEY"); key != "" {
		if r.Header.Get("Authorization") == "Bearer "+key {
			return RoleAdmin
		}
	}
	switch strings.ToLower(r.Header.Get("X-ERA-Role")) {
	case "admin":
		return RoleAdmin
	case "viewer":
		return RoleViewer
	default:
		return RoleAnalyst
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

func CanWriteCases(r Role) bool {
	return r == RoleAdmin || r == RoleAnalyst
}

func CanReadCases(r Role) bool {
	return true
}
