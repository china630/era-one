package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"era/services/platform/httpserver"
	"era/services/platform/workspace"

	"github.com/golang-jwt/jwt/v5"
)

type server struct {
	jwtSecret []byte
	licenseOK bool
	ollamaURL string
	client    *http.Client
	dialed    *bool // test hook
}

func licenseFromEnv() bool {
	if envTruthy("ERA_LICENSE_STRICT") || envTruthy("ERA_PRODUCTION") {
		return envTruthy("ERA_LICENSE_OFFICE_AI")
	}
	if envTruthy("ERA_OFFICE_DEV") {
		return true
	}
	return envTruthy("ERA_LICENSE_OFFICE_AI")
}

func envTruthy(k string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes"
}

func newMux(s *server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"docs-ai"}`))
	})
	mux.HandleFunc("/api/v1/docs-ai/summarize", s.withAuth(s.handleSummarize))
	mux.HandleFunc("/api/v1/docs-ai/rewrite", s.withAuth(s.handleRewrite))
	return mux
}

func (s *server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.licenseOK {
			http.Error(w, "office-ai license required", http.StatusForbidden)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tokStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		tok, err := jwt.Parse(tokStr, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("alg")
			}
			return s.jwtSecret, nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !tok.Valid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, _ := tok.Claims.(jwt.MapClaims)
		tid, _ := claims["tenant_id"].(string)
		sub, _ := claims["sub"].(string)
		if tid == "" || sub == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func stubSummary(text string) map[string]string {
	if len(text) > 2000 {
		text = text[:2000]
	}
	return map[string]string{
		"mode":    "stub",
		"summary": "ERA Office AI (air-gap stub): " + text,
	}
}

func stubRewrite(text string) map[string]string {
	if len(text) > 2000 {
		text = text[:2000]
	}
	return map[string]string{
		"mode":    "stub",
		"rewrite": "ERA Office AI (air-gap stub rewrite): " + text,
	}
}

// allowlistedClient dials only the host of ERA_OLLAMA_URL (air-gap).
func allowlistedClient(baseURL string, dialed *bool) (*http.Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid ERA_OLLAMA_URL")
	}
	allowedHost := u.Hostname()
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}
			if !strings.EqualFold(host, allowedHost) {
				return nil, fmt.Errorf("air-gap: dial to %s blocked (allowlist=%s)", host, allowedHost)
			}
			if dialed != nil {
				*dialed = true
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}
	return &http.Client{Timeout: 30 * time.Second, Transport: transport}, nil
}

func (s *server) handleSummarize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	text, _ := parseTextBody(r)
	out := s.summarize(text)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *server) handleRewrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	text, instruction := parseRewriteBody(r)
	out := s.rewrite(text, instruction)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func parseTextBody(r *http.Request) (string, string) {
	body, _ := io.ReadAll(r.Body)
	text := strings.TrimSpace(string(body))
	instruction := ""
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var req struct {
			Text        string `json:"text"`
			Instruction string `json:"instruction"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			if t := strings.TrimSpace(req.Text); t != "" {
				text = t
			}
			instruction = strings.TrimSpace(req.Instruction)
		}
	}
	return text, instruction
}

func parseRewriteBody(r *http.Request) (string, string) {
	return parseTextBody(r)
}

func (s *server) summarize(text string) map[string]string {
	if s.ollamaURL == "" {
		// Air-gap stub: never dial.
		return stubSummary(text)
	}
	if out, err := s.callOllama("Summarize:\n"+text, text); err == nil && out != "" {
		return map[string]string{"mode": "ollama", "summary": out}
	}
	return stubSummary(text)
}

func (s *server) rewrite(text, instruction string) map[string]string {
	if s.ollamaURL == "" {
		return stubRewrite(text)
	}
	prompt := "Rewrite:\n"
	if instruction != "" {
		prompt += instruction + "\n\n"
	}
	prompt += text
	if out, err := s.callOllama(prompt, text); err == nil && out != "" {
		return map[string]string{"mode": "ollama", "rewrite": out}
	}
	return stubRewrite(text)
}

func (s *server) callOllama(prompt, _ string) (string, error) {
	u, err := url.Parse(s.ollamaURL)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid ollama url")
	}
	endpoint := strings.TrimRight(s.ollamaURL, "/") + "/api/generate"
	payload, _ := json.Marshal(map[string]any{
		"model":  workspace.Env("ERA_OLLAMA_MODEL", "llama3.2"),
		"prompt": prompt,
		"stream": false,
	})
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := s.client
	if client == nil {
		c, err := allowlistedClient(s.ollamaURL, s.dialed)
		if err != nil {
			return "", err
		}
		client = c
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama status %d", resp.StatusCode)
	}
	var out struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.Response), nil
}

func main() {
	ollama := strings.TrimSpace(os.Getenv("ERA_OLLAMA_URL"))
	s := &server{
		jwtSecret: []byte(workspace.Env("ERA_IDENTITY_JWT_SECRET", "dev-only-change-in-prod")),
		licenseOK: licenseFromEnv(),
		ollamaURL: ollama,
	}
	addr := workspace.Env("ERA_DOCS_AI_HTTP_ADDR", ":8146")
	log.Printf("docs-ai listening %s (license=%v ollama=%v)", addr, s.licenseOK, ollama != "")
	log.Fatal(httpserver.Listen(addr, newMux(s)))
}
