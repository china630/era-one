package mail

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed web/*
var webFS embed.FS

type ctxKey int

const (
	claimsKey ctxKey = 1
	tokenKey  ctxKey = 2
)

type DriveClient interface {
	CreateAttachmentLink(tenantID, objectID string) (string, error)
}

type Server struct {
	Drive      DriveClient
	Documents  *DocumentsClient
	MailAPIURL string
	JWTSecret  []byte
}

func NewServer(d DriveClient) *Server {
	return &Server{
		Drive:      d,
		Documents:  NewDocumentsClient(env("ERA_WORKSPACE_BASE_URL", "https://app.customer.local")),
		MailAPIURL: env("ERA_MAIL_API_URL", "http://127.0.0.1:8150"),
		JWTSecret:  []byte(env("ERA_IDENTITY_JWT_SECRET", "dev-only-change-in-prod")),
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/mail/healthz", s.healthz)
	mux.HandleFunc("/mail/callback", s.callback)
	mux.HandleFunc("/mail/api/policy", s.proxyPolicy)
	mux.HandleFunc("/mail/api/mailbox", s.withJWT(s.proxyMailbox))
	mux.HandleFunc("/mail/api/message", s.withJWT(s.proxyMessage))
	mux.HandleFunc("/mail/api/messages", s.withJWT(s.proxyMessages))
	mux.HandleFunc("/mail/api/send", s.withJWT(s.proxySend))
	mux.HandleFunc("/mail/api/drive/attachment-link", s.withJWT(s.handleDriveAttachmentLink))
	mux.HandleFunc("/mail/api/documents/edit-link", s.withJWT(s.handleDocumentsEditLink))
	sub, _ := fs.Sub(webFS, "web")
	mux.Handle("/mail/static/", http.StripPrefix("/mail/static/", http.FileServer(http.FS(sub))))
	mux.HandleFunc("/mail", s.serveIndex)
	mux.HandleFunc("/mail/", s.serveIndex)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "service": "ui-mail"})
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/mail" && r.URL.Path != "/mail/" && r.URL.Path != "/mail/callback" {
		http.NotFound(w, r)
		return
	}
	b, err := webFS.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	s.serveIndex(w, r)
}

func (s *Server) withJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tok := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		claims, err := ValidateToken(tok, s.JWTSecret)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		ctx = context.WithValue(ctx, tokenKey, tok)
		next(w, r.WithContext(ctx))
	}
}

func claimsFrom(r *http.Request) *TokenClaims {
	c, _ := r.Context().Value(claimsKey).(*TokenClaims)
	return c
}

func bearerFrom(r *http.Request) string {
	tok, _ := r.Context().Value(tokenKey).(string)
	return tok
}

func forwardAuth(req *http.Request, r *http.Request, tenantID string) {
	if tok := bearerFrom(r); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if tenantID != "" {
		req.Header.Set("X-ERA-Tenant", tenantID)
	}
}

func (s *Server) proxyPolicy(w http.ResponseWriter, r *http.Request) {
	tenant := r.URL.Query().Get("tenant_id")
	resp, err := http.Get(s.MailAPIURL + "/api/v1/policy?tenant_id=" + tenant)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) proxyMailbox(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	resp, err := http.Get(s.MailAPIURL + "/api/v1/mailboxes/" + claims.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) proxyMessages(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	req, err := http.NewRequest(http.MethodGet, s.MailAPIURL+"/api/v1/mail/messages?email="+claims.Email, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	forwardAuth(req, r, claims.TenantID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) proxyMessage(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	wantUID := r.URL.Query().Get("uid")
	if wantUID == "" {
		http.Error(w, "uid required", http.StatusBadRequest)
		return
	}
	resp, err := http.Get(s.MailAPIURL + "/api/v1/mail/messages?email=" + claims.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	var payload struct {
		Messages []map[string]json.RawMessage `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	for _, m := range payload.Messages {
		var msgUID int64
		if v, ok := m["uid"]; ok {
			_ = json.Unmarshal(v, &msgUID)
		} else if v, ok := m["UID"]; ok {
			_ = json.Unmarshal(v, &msgUID)
		}
		if fmt.Sprint(msgUID) != wantUID {
			continue
		}
		subject := stringField(m, "subject", "Subject")
		body := stringField(m, "body", "Body")
		if body == "" {
			if raw := bytesField(m, "Raw", "raw"); len(raw) > 0 {
				body = extractBodyFromRaw(raw)
			}
		}
		writeJSON(w, map[string]any{
			"uid":     msgUID,
			"subject": subject,
			"body":    body,
		})
		return
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func (s *Server) proxySend(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, _ := io.ReadAll(r.Body)
	req, err := http.NewRequest(http.MethodPost, s.MailAPIURL+"/api/v1/mail/send", bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	forwardAuth(req, r, claims.TenantID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func extractBodyFromRaw(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	s := string(raw)
	if idx := strings.Index(s, "\r\n\r\n"); idx >= 0 {
		return s[idx+4:]
	}
	if idx := strings.Index(s, "\n\n"); idx >= 0 {
		return s[idx+2:]
	}
	return s
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleDocumentsEditLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.Documents == nil {
		http.Error(w, "documents not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		DriveObjectID string `json:"drive_object_id"`
		Filename      string `json:"filename"`
		ContentType   string `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !IsEradOrDocx(req.Filename, req.ContentType) {
		http.Error(w, "unsupported attachment type", http.StatusForbidden)
		return
	}
	link, err := s.Documents.EditLink(req.DriveObjectID)
	if err != nil {
		if strings.Contains(err.Error(), "license") {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"url": link, "label": "Редактировать в Documents"})
}

// handleDriveAttachmentLink — AC-C5: JWT from session → drive-api; 403 without Drive license.
func (s *Server) handleDriveAttachmentLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	claims := claimsFrom(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.Drive == nil {
		http.Error(w, "drive not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		ObjectID string `json:"object_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ObjectID == "" {
		http.Error(w, "object_id required", http.StatusBadRequest)
		return
	}
	if dc, ok := s.Drive.(*HTTPDriveClient); ok {
		dc.UserJWT = bearerFrom(r)
	}
	link, err := s.Drive.CreateAttachmentLink(claims.TenantID, req.ObjectID)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "status 403") || strings.Contains(msg, "forbidden") || strings.Contains(msg, "license") {
			http.Error(w, "drive: license denied", http.StatusForbidden)
			return
		}
		http.Error(w, msg, http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]string{"url": link})
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func stringField(m map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				return s
			}
		}
	}
	return ""
}

func bytesField(m map[string]json.RawMessage, keys ...string) []byte {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			var b []byte
			if json.Unmarshal(v, &b) == nil {
				return b
			}
			var s string
			if json.Unmarshal(v, &s) == nil {
				if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
					return decoded
				}
			}
		}
	}
	return nil
}
