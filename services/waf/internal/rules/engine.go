package rules

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
)

// Match is a fired WAF rule.
type Match struct {
	RuleID   string `json:"rule_id"`
	Category string `json:"category"`
	Severity string `json:"severity"`
}

// RuleDef is a serializable rule (JSON pack).
type RuleDef struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Severity string `json:"severity"`
	Pattern  string `json:"pattern,omitempty"`
	XXEHint  bool   `json:"xxe_hint,omitempty"`
}

type compiled struct {
	def RuleDef
	re  *regexp.Regexp
}

// Engine evaluates HTTP requests against a rule pack.
type Engine struct {
	mu    sync.RWMutex
	rules []compiled
	path  string
}

// NewOWASP loads the built-in OWASP-style pack.
func NewOWASP() *Engine {
	e := &Engine{}
	_ = e.LoadDefs(DefaultPack())
	return e
}

// DefaultPack returns the built-in rule definitions.
func DefaultPack() []RuleDef {
	return []RuleDef{
		{ID: "era-waf-sqli", Category: "A03-injection", Severity: "critical",
			Pattern: `(?i)('|\")(\s)*(or|union|select|drop|insert|delete)\s`},
		{ID: "era-waf-xss", Category: "A03-injection", Severity: "high",
			Pattern: `(?i)<\s*script|javascript:|onerror\s*=`},
		{ID: "era-waf-path-traversal", Category: "A01-broken-access", Severity: "high",
			Pattern: `(?i)(\.\./|\.\.\\|%2e%2e%2f)`},
		{ID: "era-waf-cmdi", Category: "A03-injection", Severity: "critical",
			Pattern: `(?i)(;\s*(cat|wget|curl|bash|sh)\s|&&\s*(cat|wget|curl))`},
		{ID: "era-waf-ssrf", Category: "A10-ssrf", Severity: "medium",
			Pattern: `(?i)(169\.254\.|127\.0\.0\.1|metadata\.google)`},
		{ID: "era-waf-xxe", Category: "A05-misconfig", Severity: "high", XXEHint: true},
		// CRS-lite subset (Phase 2) — not full OWASP CRS
		{ID: "era-waf-crs-rce", Category: "A03-injection", Severity: "critical",
			Pattern: `(?i)(\$\([^\)]+\)|\/etc\/passwd|cmd\.exe|powershell\.exe)`},
		{ID: "era-waf-crs-lfi", Category: "A01-broken-access", Severity: "high",
			Pattern: `(?i)(php:\/\/filter|expect:\/\/|file:\/\/\/)`},
		{ID: "era-waf-crs-ssti", Category: "A03-injection", Severity: "high",
			Pattern: `(?i)(\{\{.*\}\}|\$\{jndi:)`},
	}
}

// LoadDefs compiles and replaces rules.
func (e *Engine) LoadDefs(defs []RuleDef) error {
	out := make([]compiled, 0, len(defs))
	for _, d := range defs {
		c := compiled{def: d}
		if d.Pattern != "" {
			re, err := regexp.Compile(d.Pattern)
			if err != nil {
				return fmt.Errorf("rule %s: %w", d.ID, err)
			}
			c.re = re
		}
		out = append(out, c)
	}
	e.mu.Lock()
	e.rules = out
	e.mu.Unlock()
	return nil
}

// LoadFile loads a JSON rule pack from path.
func (e *Engine) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var defs []RuleDef
	if err := json.Unmarshal(data, &defs); err != nil {
		return err
	}
	if err := e.LoadDefs(defs); err != nil {
		return err
	}
	e.mu.Lock()
	e.path = path
	e.mu.Unlock()
	return nil
}

// Reload reloads from last path or DefaultPack.
func (e *Engine) Reload() error {
	e.mu.RLock()
	path := e.path
	e.mu.RUnlock()
	if path == "" {
		return e.LoadDefs(DefaultPack())
	}
	return e.LoadFile(path)
}

// Rules returns current rule defs.
func (e *Engine) Rules() []RuleDef {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]RuleDef, len(e.rules))
	for i, r := range e.rules {
		out[i] = r.def
	}
	return out
}

func compileRule(d RuleDef) (compiled, error) {
	c := compiled{def: d}
	if d.Pattern != "" {
		re, err := regexp.Compile(d.Pattern)
		if err != nil {
			return c, fmt.Errorf("rule %s: %w", d.ID, err)
		}
		c.re = re
	}
	return c, nil
}

// AddRule appends (or replaces by id) an in-memory rule. Reload() still restores from file/pack.
func (e *Engine) AddRule(d RuleDef) error {
	if d.ID == "" {
		return fmt.Errorf("rule id required")
	}
	c, err := compileRule(d)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, existing := range e.rules {
		if existing.def.ID == d.ID {
			e.rules[i] = c
			return nil
		}
	}
	e.rules = append(e.rules, c)
	return nil
}

// UpdateRule replaces a rule by id.
func (e *Engine) UpdateRule(id string, d RuleDef) error {
	if id == "" {
		return fmt.Errorf("rule id required")
	}
	d.ID = id
	c, err := compileRule(d)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, existing := range e.rules {
		if existing.def.ID == id {
			e.rules[i] = c
			return nil
		}
	}
	return fmt.Errorf("not found")
}

// DeleteRule removes a rule by id.
func (e *Engine) DeleteRule(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, existing := range e.rules {
		if existing.def.ID == id {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return true
		}
	}
	return false
}

// GetRule returns a rule by id.
func (e *Engine) GetRule(id string) (RuleDef, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, r := range e.rules {
		if r.def.ID == id {
			return r.def, true
		}
	}
	return RuleDef{}, false
}

// Evaluate returns a match when the request should be blocked.
func (e *Engine) Evaluate(r *http.Request) (Match, bool) {
	return e.EvaluateTarget(buildTarget(r, ""))
}

// EvaluateWithBody includes POST body (truncated) in the scan target.
func (e *Engine) EvaluateWithBody(r *http.Request, body string) (Match, bool) {
	return e.EvaluateTarget(buildTarget(r, body))
}

func buildTarget(r *http.Request, body string) string {
	target := r.URL.Path
	if q, err := url.QueryUnescape(r.URL.RawQuery); err == nil {
		target += " " + q
	} else {
		target += " " + r.URL.RawQuery
	}
	for _, h := range []string{"User-Agent", "Cookie", "Referer"} {
		target += " " + r.Header.Get(h)
	}
	if body != "" {
		target += " " + body
	}
	return target
}

func (e *Engine) EvaluateTarget(target string) (Match, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, rule := range e.rules {
		if rule.re != nil && rule.re.MatchString(target) {
			return Match{RuleID: rule.def.ID, Category: rule.def.Category, Severity: rule.def.Severity}, true
		}
		if rule.def.XXEHint && strings.Contains(target, "ENTITY") && strings.Contains(strings.ToLower(target), "xml") {
			return Match{RuleID: rule.def.ID, Category: rule.def.Category, Severity: rule.def.Severity}, true
		}
	}
	return Match{}, false
}
