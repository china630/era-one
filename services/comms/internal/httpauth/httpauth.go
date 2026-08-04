// Package httpauth — Comms API AuthZ (G0): JWT Bearer, Basic, internal token, DEV bypass.
// Prod: no header-trust without JWT/Basic password. Dev: ERA_*_DEV=1 allows X-ERA-* / Basic.
package httpauth

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey int

const principalKey ctxKey = 1

// Principal — authenticated caller.
type Principal struct {
	TenantID string
	UserID   string
	Roles    []string
	Mode     string // jwt | basic | internal | dev
}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

func withPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// Config from env.
type Config struct {
	JWTSecret     []byte
	InternalToken string
	DevEnvKey     string // e.g. ERA_MAIL_DEV — when "1", header trust allowed
	RequiredRole  string // substring match in roles/claims, empty = any auth
}

// FromEnv builds config. jwtSecretEnv defaults ERA_IDENTITY_JWT_SECRET.
func FromEnv(devKey, requiredRole string) Config {
	return Config{
		JWTSecret:     []byte(os.Getenv("ERA_IDENTITY_JWT_SECRET")),
		InternalToken: os.Getenv("ERA_INTERNAL_TOKEN"),
		DevEnvKey:     devKey,
		RequiredRole:  requiredRole,
	}
}

func (c Config) DevEnabled() bool {
	return c.DevEnvKey != "" && os.Getenv(c.DevEnvKey) == "1"
}

// Wrap protects handler. Open paths should not use Wrap.
func (c Config) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := c.Authenticate(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if c.RequiredRole != "" && !hasRole(p.Roles, c.RequiredRole) && p.Mode != "internal" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r.WithContext(withPrincipal(r.Context(), p)))
	}
}

// WrapHandler for http.Handler.
func (c Config) WrapHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := c.Authenticate(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if c.RequiredRole != "" && !hasRole(p.Roles, c.RequiredRole) && p.Mode != "internal" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), p)))
	})
}

// RequireInternal only accepts ERA_INTERNAL_TOKEN (or DEV bypass).
func (c Config) RequireInternal(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.InternalToken != "" && internalOK(r, c.InternalToken) {
			next(w, r.WithContext(withPrincipal(r.Context(), Principal{Mode: "internal", TenantID: "internal"})))
			return
		}
		if c.DevEnabled() {
			log.Printf("httpauth: DEV bypass internal path %s (set %s=1)", r.URL.Path, c.DevEnvKey)
			next(w, r.WithContext(withPrincipal(r.Context(), Principal{Mode: "dev", TenantID: "dev"})))
			return
		}
		if c.InternalToken == "" && c.DevEnabled() {
			next(w, r)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	}
}

func (c Config) RequireInternalHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(c.RequireInternal(next.ServeHTTP))
}

// Authenticate resolves JWT, HTTP Basic, internal token, or DEV headers.
// Fail-closed: Basic needs DEV or ERA_PROTOCOL_BASIC_PASSWORD match; else Bearer JWT.
func (c Config) Authenticate(r *http.Request) (Principal, error) {
	auth := r.Header.Get("Authorization")
	tok := ""
	if strings.HasPrefix(auth, "Bearer ") {
		tok = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if tok == "" {
		tok = strings.TrimSpace(r.Header.Get("X-ERA-Internal-Token"))
	}

	if c.InternalToken != "" && tok == c.InternalToken {
		return Principal{Mode: "internal", TenantID: headerOr(r, "X-ERA-Tenant", "internal"), UserID: "service"}, nil
	}

	if strings.HasPrefix(auth, "Basic ") {
		if p, ok := c.authenticateBasic(r, strings.TrimSpace(strings.TrimPrefix(auth, "Basic "))); ok {
			return p, nil
		}
	}

	if tok != "" && len(c.JWTSecret) > 0 {
		p, err := parseJWT(tok, c.JWTSecret)
		if err == nil {
			return p, nil
		}
	}

	if c.DevEnabled() {
		tid := strings.TrimSpace(r.Header.Get("X-ERA-Tenant"))
		role := strings.TrimSpace(r.Header.Get("X-ERA-Role"))
		uid := strings.TrimSpace(r.Header.Get("X-ERA-User"))
		if tid == "" {
			tid = "t-demo"
		}
		if uid == "" {
			uid = "dev-user"
		}
		log.Printf("httpauth: DEV bypass mode=dev path=%s", r.URL.Path)
		roles := []string{}
		if role != "" {
			roles = strings.FieldsFunc(role, func(r rune) bool { return r == ',' || r == ' ' })
		}
		return Principal{Mode: "dev", TenantID: tid, UserID: uid, Roles: roles}, nil
	}

	return Principal{}, errUnauthorized
}

// authenticateBasic decodes Basic credentials. Returns ok=false to fall through.
func (c Config) authenticateBasic(r *http.Request, b64 string) (Principal, bool) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return Principal{}, false
	}
	user, pass, ok := strings.Cut(string(raw), ":")
	if !ok {
		user = string(raw)
		pass = ""
	}
	user = strings.TrimSpace(user)
	if user == "" {
		return Principal{}, false
	}

	// JWT-looking username (header.payload.sig): try as token, skip Basic principal.
	if len(c.JWTSecret) > 0 && looksLikeJWT(user) {
		if p, err := parseJWT(user, c.JWTSecret); err == nil {
			return p, true
		}
		return Principal{}, false
	}

	tid := headerOr(r, "X-ERA-Tenant", "t-demo")
	if c.DevEnabled() {
		// DEV: non-empty user is enough (Outlook/Thunderbird lab Basic).
		return Principal{Mode: "basic", TenantID: tid, UserID: user}, true
	}
	want := os.Getenv("ERA_PROTOCOL_BASIC_PASSWORD")
	if want != "" && pass == want {
		return Principal{Mode: "basic", TenantID: tid, UserID: user}, true
	}
	return Principal{}, false
}

func looksLikeJWT(s string) bool {
	return strings.Count(s, ".") == 2 && len(s) > 20
}

var errUnauthorized = errAuth("unauthorized")

type errAuth string

func (e errAuth) Error() string { return string(e) }

func parseJWT(tokStr string, secret []byte) (Principal, error) {
	tok, err := jwt.Parse(tokStr, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errAuth("alg")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		return Principal{}, errAuth("invalid jwt")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return Principal{}, errAuth("claims")
	}
	tenantID, _ := claims["tenant_id"].(string)
	sub, _ := claims["sub"].(string)
	if tenantID == "" || sub == "" {
		return Principal{}, errAuth("missing tenant_id/sub")
	}
	var roles []string
	if raw, ok := claims["roles"].([]any); ok {
		for _, g := range raw {
			if s, ok := g.(string); ok {
				roles = append(roles, s)
			}
		}
	}
	if raw, ok := claims["role"].(string); ok && raw != "" {
		roles = append(roles, raw)
	}
	if g, ok := claims["groups"].([]any); ok {
		for _, x := range g {
			if s, ok := x.(string); ok {
				roles = append(roles, s)
			}
		}
	}
	return Principal{Mode: "jwt", TenantID: tenantID, UserID: sub, Roles: roles}, nil
}

func internalOK(r *http.Request, want string) bool {
	if want == "" {
		return false
	}
	if r.Header.Get("X-ERA-Internal-Token") == want {
		return true
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") && strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")) == want {
		return true
	}
	return false
}

func hasRole(roles []string, need string) bool {
	need = strings.TrimSpace(need)
	if need == "" {
		return true
	}
	for _, r := range roles {
		if strings.Contains(r, need) || r == need {
			return true
		}
	}
	return false
}

func headerOr(r *http.Request, k, def string) string {
	if v := strings.TrimSpace(r.Header.Get(k)); v != "" {
		return v
	}
	return def
}

// MintDevJWT — test helper HS256 token.
func MintDevJWT(secret []byte, tenantID, sub, role string) (string, error) {
	claims := jwt.MapClaims{
		"tenant_id": tenantID,
		"sub":       sub,
		"roles":     []string{role},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}
