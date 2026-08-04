package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"era/services/platform/envelope"
	"era/services/platform/licensegate"
	"era/services/waf/internal/rules"
)

// Server is the WAF HTTP surface.
type Server struct {
	Engine    *rules.Engine
	Upstream  *url.URL
	Gate      *licensegate.Gate
	Pub       *envelope.Publisher
	BodyLimit int64
	Proxy     *httputil.ReverseProxy
}

// New builds a WAF server.
func New(eng *rules.Engine, upstream string, gate *licensegate.Gate, pub *envelope.Publisher, bodyLimit int64) (*Server, error) {
	if bodyLimit <= 0 {
		bodyLimit = 1 << 20
	}
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, e error) {
		http.Error(w, e.Error(), http.StatusBadGateway)
	}
	proxy.ModifyResponse = nil
	s := &Server{Engine: eng, Upstream: u, Gate: gate, Pub: pub, BodyLimit: bodyLimit, Proxy: proxy}
	return s, nil
}

// Routes returns the mux.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/waf/rules", s.handleRules)
	mux.HandleFunc("/api/v1/waf/rules/", s.handleRuleByID)
	mux.HandleFunc("/api/v1/waf/reload", s.handleReload)
	mux.HandleFunc("/", s.handleProxy)
	return mux
}

func (s *Server) requirePerimeter(w http.ResponseWriter) bool {
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModulePerimeter) {
		http.Error(w, `{"error":"module perimeter not licensed"}`, http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	if !s.requirePerimeter(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"rules": s.Engine.Rules()})
	case http.MethodPost:
		var def rules.RuleDef
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil || def.ID == "" {
			http.Error(w, "id and rule body required", http.StatusBadRequest)
			return
		}
		if err := s.Engine.AddRule(def); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, def)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRuleByID(w http.ResponseWriter, r *http.Request) {
	if !s.requirePerimeter(w) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/waf/rules/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var def rules.RuleDef
		if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := s.Engine.UpdateRule(id, def); err != nil {
			if err.Error() == "not found" {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		got, _ := s.Engine.GetRule(id)
		writeJSON(w, http.StatusOK, got)
	case http.MethodDelete:
		if !s.Engine.DeleteRule(id) {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
	case http.MethodGet:
		got, ok := s.Engine.GetRule(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, got)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requirePerimeter(w) {
		return
	}
	if err := s.Engine.Reload(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reloaded": true, "count": len(s.Engine.Rules())})
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	if !s.requirePerimeter(w) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.BodyLimit)
	var bodyStr string
	if r.Body != nil && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		bodyStr = string(b)
		r.Body = io.NopCloser(bytes.NewReader(b))
	}
	if m, blocked := s.Engine.EvaluateWithBody(r, bodyStr); blocked {
		log.Printf("WAF BLOCK rule=%s cat=%s path=%s", m.RuleID, m.Category, r.URL.Path)
		if s.Pub != nil {
			_ = s.Pub.PublishNetwork(context.Background(), r.RemoteAddr, s.Upstream.Host, "http", "blocked:"+m.RuleID, 0)
		}
		writeJSON(w, http.StatusForbidden, map[string]any{
			"blocked": true, "rule_id": m.RuleID, "category": m.Category, "severity": m.Severity,
		})
		return
	}
	s.Proxy.ServeHTTP(w, r)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// IdleTimeout for reverse proxy transport.
func DefaultTransport() *http.Transport {
	return &http.Transport{ResponseHeaderTimeout: 30 * time.Second}
}

// DrainBody helper for tests.
func DrainBody(r io.ReadCloser) {
	if r != nil {
		_, _ = io.Copy(io.Discard, r)
		_ = r.Close()
	}
}

// ParseBodyLimit parses ERA_WAF_BODY_LIMIT.
func ParseBodyLimit(s string) int64 {
	if s == "" {
		return 1 << 20
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 1 << 20
	}
	return n
}
