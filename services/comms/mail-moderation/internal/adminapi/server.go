// Package adminapi — CRUD rules, YAML, action links, HR, Admin UI.
package adminapi

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"era/services/comms/internal/httpauth"
	"era/services/comms/mail-moderation/internal/engine"
	"era/services/comms/mail-moderation/internal/hold"
	"era/services/comms/mail-moderation/internal/notify"
	"era/services/comms/mail-moderation/internal/policy"
	"era/services/comms/mail-moderation/internal/resolve"
	"era/services/comms/mail-moderation/internal/rules"
)

// Templates — встроенные шаблоны (PRD A-03).
func Templates() map[string]policy.Document {
	on := true
	return map[string]policy.Document{
		"new-hires-external": {Rules: []policy.Rule{{
			ID: "new-hires-external", Priority: 10, StopProcessing: true,
			Conditions: policy.Conditions{SenderGroups: []string{"novices"}, ExternalOnly: true, ExcludeSystem: true},
			Moderator:  policy.ModeratorSpec{Mode: policy.ModManager, Fallback: []string{"hr-moderation@company.local"}},
			TTLHours: 72, NotifyOnHold: &on,
		}}},
		"vip-domain": {Rules: []policy.Rule{{
			ID: "vip-domain", Priority: 20, StopProcessing: true,
			Conditions: policy.Conditions{RecipientDomains: []string{"vip-customer.com"}},
			Moderator:  policy.ModeratorSpec{Mode: policy.ModLDAPAttr, LDAPAttr: "extensionAttribute1", Fallback: []string{"vip-desk@company.local"}},
		}}},
		"keyword-legal": {Rules: []policy.Rule{{
			ID: "keyword-legal", Priority: 15, StopProcessing: true,
			Conditions: policy.Conditions{Keywords: []string{"Договор", "Смета"}, ExternalOnly: true},
			Moderator:  policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"legal@company.local"}},
		}}},
		"moderated-dl": {Rules: []policy.Rule{{
			ID: "dl-finance", Priority: 1, StopProcessing: true,
			Conditions: policy.Conditions{ModeratedRecipients: []string{"finance@company.local"}},
			Moderator:  policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"cfo@company.local"}},
			TTLHours: 48, NotifyOnHold: &on,
		}}},
	}
}

// RulesPersist — optional PG/memory rules save.
type RulesPersist interface {
	SaveDocument(doc policy.Document) error
	LoadDocument() (policy.Document, error)
}

// Server — HTTP API :8360.
type Server struct {
	mu       sync.Mutex
	Rules    []policy.Rule
	Engine   *engine.Engine
	Tokens   *notify.Tokens
	Persist  RulesPersist
	Curators *resolve.PGCurators
	// Novices — in-memory group membership for HR API when no LDAP JSON.
	Novices map[string]string // sender → curator
}

func New(eng *engine.Engine, tokens *notify.Tokens) *Server {
	s := &Server{Engine: eng, Tokens: tokens, Novices: map[string]string{}, Persist: &rules.MemorySave{}}
	if eng != nil {
		s.Rules = append([]policy.Rule(nil), eng.Rules...)
	}
	return s
}

func (s *Server) adminAuth() httpauth.Config {
	// Lab: ERA_MM_DEV or ERA_MAIL_DEV; prod: JWT/internal + mm.admin (fail-closed).
	if os.Getenv("ERA_MM_DEV") != "1" && os.Getenv("ERA_MAIL_DEV") == "1" {
		return httpauth.FromEnv("ERA_MAIL_DEV", "mm.admin")
	}
	return httpauth.FromEnv("ERA_MM_DEV", "mm.admin")
}

func (s *Server) Register(mux *http.ServeMux) {
	auth := s.adminAuth()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// Admin surface — httpauth required (mm.admin). Token action links stay public.
	mux.HandleFunc("/v1/moderation/rules", auth.Wrap(s.handleRules))
	mux.HandleFunc("/v1/moderation/rules/import", auth.Wrap(s.handleImport))
	mux.HandleFunc("/v1/moderation/rules/export", auth.Wrap(s.handleExport))
	mux.HandleFunc("/v1/moderation/templates", auth.Wrap(s.handleTemplates))
	mux.HandleFunc("/v1/moderation/action", s.handleAction)
	mux.HandleFunc("/v1/moderation/holds", auth.Wrap(s.handleHoldsList))
	mux.HandleFunc("/v1/moderation/holds/", auth.Wrap(s.handleForceRelease))
	mux.HandleFunc("/v1/moderation/hr/novices", auth.Wrap(s.handleHRNovices))
	mux.HandleFunc("/v1/moderation/simulate", auth.Wrap(s.handleSimulate))
	mux.HandleFunc("/ui/", auth.Wrap(s.handleUI))
	mux.HandleFunc("/ui", auth.Wrap(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusFound)
	}))
}

func (s *Server) syncEngine() {
	if s.Engine != nil {
		s.Engine.Rules = append([]policy.Rule(nil), s.Rules...)
	}
	if s.Persist != nil {
		_ = s.Persist.SaveDocument(policy.Document{Rules: s.Rules})
	}
}

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, policy.Document{Rules: s.Rules})
	case http.MethodPut:
		var doc policy.Document
		if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := policy.ValidateDocument(doc); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		s.Rules = doc.Rules
		s.syncEngine()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method", 405)
	}
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	doc, err := policy.ParseDocument(b)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	s.mu.Lock()
	s.Rules = doc.Rules
	s.syncEngine()
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", 405)
		return
	}
	s.mu.Lock()
	doc := policy.Document{Rules: s.Rules}
	s.mu.Unlock()
	b, err := policy.MarshalDocument(doc)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(b)
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", 405)
		return
	}
	writeJSON(w, Templates())
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	comment := r.URL.Query().Get("comment")
	if token == "" {
		http.Error(w, "token required", 400)
		return
	}
	holdID, mod, action, err := s.Tokens.Consume(token)
	if err != nil {
		http.Error(w, err.Error(), 403)
		return
	}
	if action == "reject" && comment == "" {
		comment = r.FormValue("comment")
	}
	if err := s.Engine.ApplyAction(holdID, mod, action, comment); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(action + " ok"))
}

func (s *Server) handleForceRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	// AuthZ + mm.admin enforced by Register Wrap (RequiredRole).
	mod := "admin"
	if p, ok := httpauth.FromContext(r.Context()); ok && p.UserID != "" {
		mod = p.UserID
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/moderation/holds/")
	if id == "" || id == r.URL.Path {
		http.Error(w, "id", 400)
		return
	}
	action := r.URL.Query().Get("action")
	if action == "" {
		action = "approve"
	}
	if err := s.Engine.ApplyAction(id, mod, action, r.URL.Query().Get("comment")); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleHoldsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", 405)
		return
	}
	var list []hold.Record
	if l, ok := s.Engine.Holds.(hold.Lister); ok {
		list = l.ListPending()
	}
	writeJSON(w, list)
}

type hrNoviceReq struct {
	Sender  string `json:"sender"`
	Curator string `json:"curator"`
}

func (s *Server) handleHRNovices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	var req hrNoviceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	req.Sender = strings.ToLower(strings.TrimSpace(req.Sender))
	req.Curator = strings.TrimSpace(req.Curator)
	if req.Sender == "" || req.Curator == "" {
		http.Error(w, "sender and curator required", 400)
		return
	}
	s.mu.Lock()
	s.Novices[req.Sender] = req.Curator
	s.mu.Unlock()
	if s.Curators != nil {
		_ = s.Curators.Set(req.Sender, req.Curator)
	}
	// Enrich engine groups if StaticGroups
	if sg, ok := s.Engine.Groups.(engine.StaticGroups); ok {
		g := append([]string{}, sg[req.Sender]...)
		has := false
		for _, x := range g {
			if strings.EqualFold(x, "novices") {
				has = true
			}
		}
		if !has {
			g = append(g, "novices")
		}
		sg[req.Sender] = g
		s.Engine.Groups = sg
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	var msg policy.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	s.mu.Lock()
	rules := append([]policy.Rule(nil), s.Rules...)
	s.mu.Unlock()
	local := []string{}
	if s.Engine != nil {
		local = s.Engine.Local
	}
	res := policy.Evaluate(rules, msg, policy.EvalContext{LocalDomains: local})
	writeJSON(w, res)
}

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	nRules := len(s.Rules)
	s.mu.Unlock()
	var pending int
	if s.Engine != nil {
		if l, ok := s.Engine.Holds.(hold.Lister); ok {
			pending = len(l.ListPending())
		}
	}
	_ = uiTmpl.Execute(w, map[string]any{"Rules": nRules, "Pending": pending})
}

var uiTmpl = template.Must(template.New("ui").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>ERA Mail Moderation</title>
<style>body{font-family:system-ui;margin:2rem;max-width:42rem}code{background:#f4f4f4;padding:.1rem .3rem}pre{background:#f8f8f8;padding:.75rem;overflow:auto}</style>
</head><body>
<h1>ERA Mail Moderation</h1>
<p>Rules loaded: <b>{{.Rules}}</b> · Pending holds: <b>{{.Pending}}</b></p>
<ul>
<li><code>GET/PUT /v1/moderation/rules</code></li>
<li><code>POST /v1/moderation/rules/import</code> (YAML)</li>
<li><code>GET /v1/moderation/rules/export</code></li>
<li><code>POST /v1/moderation/simulate</code> (JSON Message)</li>
<li><code>GET /v1/moderation/holds</code></li>
<li><code>POST /v1/moderation/holds/{id}?action=approve|reject|escalate</code> — force-release / escalate (mm.admin JWT)</li>
<li><code>POST /v1/moderation/hr/novices</code> {"sender","curator"}</li>
</ul>
<h2>Lab: force-release / escalate</h2>
<pre>curl -X POST -H "Authorization: Bearer $MM_ADMIN_JWT" \
  "$MM_API/v1/moderation/holds/$HOLD_ID?action=approve"
curl -X POST -H "Authorization: Bearer $MM_ADMIN_JWT" \
  "$MM_API/v1/moderation/holds/$HOLD_ID?action=escalate&amp;comment=L2"</pre>
<p>IceWarp milter field path remains paused (partner). Native SMTP/milter lab only.</p>
</body></html>`))

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
