package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"era/services/platform/licensegate"
	"era/services/resolve/internal/atlas"
	"era/services/resolve/internal/dnsx"
	"era/services/resolve/internal/doh"
	"era/services/resolve/internal/guard"
	"era/services/resolve/internal/policy"
	"era/services/resolve/internal/settings"
	"era/services/resolve/internal/trace"
	"github.com/google/uuid"
)

// Server is the Resolve HTTP API.
type Server struct {
	Guard    *guard.Engine
	Policy   *policy.Store
	Atlas    *atlas.Store
	Trace    *trace.Buffer
	Gate     *licensegate.Gate
	Settings *settings.Settings
	DNS      *dnsx.Server
	UIDir    string
}

func New(g *guard.Engine, pol *policy.Store, atl *atlas.Store, tr *trace.Buffer, gate *licensegate.Gate) *Server {
	return &Server{Guard: g, Policy: pol, Atlas: atl, Trace: tr, Gate: gate, Settings: settings.New()}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok", "edition": "resolve",
			"doh_enabled": s.Settings != nil && s.Settings.DoH(),
			"pack":        s.Atlas.Meta(),
		})
	})
	mux.HandleFunc("/api/v1/resolve/verdict", s.handleVerdict)
	mux.HandleFunc("/api/v1/resolve/policy", s.handlePolicy)
	mux.HandleFunc("/api/v1/resolve/policy/rules", s.handlePolicyRules)
	mux.HandleFunc("/api/v1/resolve/policy/rules/", s.handlePolicyRuleByID)
	mux.HandleFunc("/api/v1/resolve/packs", s.handlePacks)
	mux.HandleFunc("/api/v1/resolve/packs/", s.handlePackByID)
	mux.HandleFunc("/api/v1/resolve/packs/reload", s.handlePackReload)
	mux.HandleFunc("/api/v1/resolve/trace", s.handleTrace)
	mux.HandleFunc("/api/v1/resolve/settings", s.handleSettings)
	dohH := &doh.Handler{
		DNS: s.DNS,
		Enabled: func() bool {
			if s.Settings == nil {
				return true
			}
			return s.Settings.DoH()
		},
	}
	mux.Handle("/dns-query", dohH)
	if s.UIDir != "" {
		mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir(s.UIDir))))
	}
	return mux
}

func (s *Server) requireResolve(w http.ResponseWriter) bool {
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModuleResolve) {
		http.Error(w, `{"error":"module resolve not licensed"}`, http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) handleVerdict(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireResolve(w) {
		return
	}
	var req struct {
		QName string `json:"qname"`
		QType string `json:"qtype"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.QName == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.QType == "" {
		req.QType = "A"
	}
	v := s.Guard.Decide(req.QName, req.QType)
	if s.Trace != nil {
		s.Trace.Record(v)
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handlePolicy(w http.ResponseWriter, r *http.Request) {
	if !s.requireResolve(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"rules": s.Policy.List()})
	case http.MethodPost:
		var rules []policy.Rule
		if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		s.Policy.Replace(rules)
		writeJSON(w, http.StatusOK, map[string]any{"saved": true, "count": len(rules)})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePolicyRules(w http.ResponseWriter, r *http.Request) {
	if !s.requireResolve(w) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var rule policy.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if rule.Action == "" {
		http.Error(w, "action required", http.StatusBadRequest)
		return
	}
	if rule.Domain == "" && rule.Suffix == "" {
		http.Error(w, "domain or suffix required", http.StatusBadRequest)
		return
	}
	if rule.ID == "" {
		rule.ID = uuid.NewString()
	}
	s.Policy.Add(rule)
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handlePolicyRuleByID(w http.ResponseWriter, r *http.Request) {
	if !s.requireResolve(w) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/resolve/policy/rules/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		rule, ok := s.Policy.Get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, rule)
	case http.MethodDelete:
		if !s.Policy.Delete(id) {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePacks(w http.ResponseWriter, r *http.Request) {
	if !s.requireResolve(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.Atlas.Meta())
	case http.MethodPost:
		var p atlas.Pack
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.Atlas.Load(p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"loaded": true, "id": p.ID, "domains": len(p.Domains)})
	case http.MethodDelete:
		meta := s.Atlas.Meta()
		if !s.Atlas.Clear("") {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": meta.ID})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePackByID(w http.ResponseWriter, r *http.Request) {
	if !s.requireResolve(w) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/resolve/packs/")
	if id == "" || id == "reload" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Atlas.Clear(id) {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}

func (s *Server) handlePackReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireResolve(w) {
		return
	}
	dir := os.Getenv("ERA_ATLAS_PACK_DIR")
	if dir == "" {
		dir = os.Getenv("ERA_RESOLVE_ATLAS_PATH")
	}
	var req struct {
		Dir  string `json:"dir,omitempty"`
		Path string `json:"path,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Dir != "" {
		dir = req.Dir
	}
	if req.Path != "" {
		if err := s.Atlas.LoadFile(req.Path); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reloaded": true, "source": req.Path, "meta": s.Atlas.Meta()})
		return
	}
	if dir == "" {
		http.Error(w, "set ERA_ATLAS_PACK_DIR or body.dir/path (USB air-gap)", http.StatusBadRequest)
		return
	}
	st, err := os.Stat(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !st.IsDir() {
		if err := s.Atlas.LoadFile(dir); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reloaded": true, "source": dir, "meta": s.Atlas.Meta()})
		return
	}
	meta, err := s.Atlas.LoadFromDir(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reloaded": true, "source": dir, "meta": meta})
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if !s.requireResolve(w) {
		return
	}
	if s.Settings == nil {
		s.Settings = settings.New()
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.Settings.Get())
	case http.MethodPost:
		var req struct {
			DoHEnabled *bool `json:"doh_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.DoHEnabled != nil {
			s.Settings.SetDoH(*req.DoHEnabled)
		}
		writeJSON(w, http.StatusOK, s.Settings.Get())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTrace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireResolve(w) {
		return
	}
	q := r.URL.Query().Get("q")
	writeJSON(w, http.StatusOK, map[string]any{"recent": s.Trace.Filter(50, q), "q": q})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// ResolveUIDir finds ui/resolve relative to cwd or ERA_UI_DIR.
func ResolveUIDir() string {
	if d := os.Getenv("ERA_UI_DIR"); d != "" {
		return d
	}
	candidates := []string{
		"ui/resolve",
		filepath.Join("..", "..", "ui", "resolve"),
		filepath.Join("..", "..", "..", "ui", "resolve"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return ""
}
