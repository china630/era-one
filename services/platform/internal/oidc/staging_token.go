package oidc

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type stagingTokenReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// registerStaging mounts dev-only machine token + register endpoints (lab / TE).
func (s *Server) registerStaging(mux *http.ServeMux) {
	if os.Getenv("ERA_IDENTITY_DEV") != "1" {
		return
	}
	mux.HandleFunc("/oauth2/staging/token", s.stagingToken)
	mux.HandleFunc("/oauth2/staging/register", s.stagingRegister)
}

type stagingRegisterReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (s *Server) stagingRegister(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ERA_IDENTITY_DEV") != "1" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req stagingRegisterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		http.Error(w, "password too short", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = email
	}
	u, err := s.store.CreateUser("t-demo", email, name, req.Password)
	if err != nil {
		if err == ErrUserExists {
			http.Error(w, "user exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"id":        u.ID,
		"email":     u.Email,
		"tenant_id": u.TenantID,
	})
}

func (s *Server) stagingToken(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ERA_IDENTITY_DEV") != "1" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req stagingTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}
	if !s.store.VerifyLogin(email, req.Password) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	u, ok := s.store.GetUserByEmail(email)
	if !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	access, err := s.mintAccessToken(u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}

func (s *Server) mintAccessToken(u User) (string, error) {
	claims := jwt.MapClaims{
		"sub":       u.ID,
		"email":     u.Email,
		"tenant_id": u.TenantID,
		"roles":     u.Roles,
		"iss":       s.issuer,
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(s.secret)
}
