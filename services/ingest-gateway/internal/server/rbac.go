// Package server — RBAC propagation для internal HTTP (S-04).
package server

import (
	"context"
	"net/http"
	"os"
)

type principalCtxKey struct{}

// PrincipalFromContext возвращает X-ERA-Principal из контекста запроса.
func PrincipalFromContext(ctx context.Context) string {
	p, _ := ctx.Value(principalCtxKey{}).(string)
	return p
}

// RBACStrictFromEnv — true при ERA_RBAC_STRICT=1.
func RBACStrictFromEnv() bool {
	return os.Getenv("ERA_RBAC_STRICT") == "1"
}

func rbacExempt(path string) bool {
	switch path {
	case "/healthz", "/readyz", "/metrics":
		return true
	default:
		return false
	}
}

// WithRBAC оборачивает handler: требует X-ERA-Principal при strict, прокидывает в контекст.
func WithRBAC(strict bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rbacExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		principal := r.Header.Get("X-ERA-Principal")
		if principal == "" && strict {
			writeJSON(w, http.StatusUnauthorized, statusResponse{Status: "missing X-ERA-Principal"})
			return
		}
		if principal == "" {
			principal = "anonymous"
		}
		w.Header().Set("X-ERA-Principal", principal)
		ctx := context.WithValue(r.Context(), principalCtxKey{}, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
