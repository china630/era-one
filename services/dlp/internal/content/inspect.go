// Package content — file/path content DLP (Phase 2). Session DLP stays in session/.
package content

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Finding is a single DLP hit.
type Finding struct {
	RuleID   string `json:"rule_id"`
	Category string `json:"category"`
	Severity string `json:"severity"`
	Snippet  string `json:"snippet,omitempty"`
}

// Rule matches MIME, path, or body keywords.
type Rule struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Severity string `json:"severity"`
	MIME     string `json:"mime,omitempty"`
	PathGlob string `json:"path_glob,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
	re       *regexp.Regexp
}

// Engine holds content rules.
type Engine struct {
	Rules []Rule
}

// Default returns keyword/MIME/path pack.
func Default() *Engine {
	defs := []Rule{
		{ID: "era-dlp-ssn", Category: "pii", Severity: "high", Pattern: `\b\d{3}-\d{2}-\d{4}\b`},
		{ID: "era-dlp-cc", Category: "pci", Severity: "critical", Pattern: `\b(?:\d[ -]*?){13,16}\b`},
		{ID: "era-dlp-secret", Category: "secret", Severity: "critical", Pattern: `(?i)(api[_-]?key|password)\s*[:=]\s*\S+`},
		{ID: "era-dlp-pem", Category: "secret", Severity: "high", Pattern: `-----BEGIN (RSA |EC )?PRIVATE KEY-----`},
		{ID: "era-dlp-mime-exe", Category: "malware", Severity: "medium", MIME: "application/x-msdownload"},
		{ID: "era-dlp-path-secrets", Category: "path", Severity: "high", PathGlob: "**/*secret*"},
	}
	e := &Engine{}
	_ = e.Load(defs)
	return e
}

// Load compiles patterns.
func (e *Engine) Load(defs []Rule) error {
	out := make([]Rule, 0, len(defs))
	for _, d := range defs {
		if d.Pattern != "" {
			re, err := regexp.Compile(d.Pattern)
			if err != nil {
				return err
			}
			d.re = re
		}
		out = append(out, d)
	}
	e.Rules = out
	return nil
}

// Request is an inspect payload.
type Request struct {
	Path    string `json:"path,omitempty"`
	MIME    string `json:"mime,omitempty"`
	Content string `json:"content,omitempty"`
}

// Result of inspection.
type Result struct {
	Blocked  bool      `json:"blocked"`
	Findings []Finding `json:"findings"`
}

// Inspect evaluates path/MIME/content.
func (e *Engine) Inspect(req Request) Result {
	var findings []Finding
	base := filepath.ToSlash(req.Path)
	for _, r := range e.Rules {
		hit := false
		snip := ""
		if r.MIME != "" && strings.EqualFold(r.MIME, req.MIME) {
			hit = true
			snip = "mime:" + req.MIME
		}
		if r.PathGlob != "" && pathMatch(r.PathGlob, base) {
			hit = true
			snip = "path:" + base
		}
		if r.re != nil && req.Content != "" {
			if loc := r.re.FindStringIndex(req.Content); loc != nil {
				hit = true
				start, end := loc[0], loc[1]
				if end-start > 64 {
					end = start + 64
				}
				snip = req.Content[start:end]
			}
		}
		if hit {
			findings = append(findings, Finding{RuleID: r.ID, Category: r.Category, Severity: r.Severity, Snippet: snip})
		}
	}
	blocked := false
	for _, f := range findings {
		if f.Severity == "critical" || f.Severity == "high" {
			blocked = true
			break
		}
	}
	return Result{Blocked: blocked, Findings: findings}
}

func pathMatch(glob, path string) bool {
	g := strings.TrimPrefix(glob, "**/")
	g = strings.ToLower(g)
	p := strings.ToLower(path)
	if strings.Contains(g, "*") {
		g = strings.ReplaceAll(g, "*", "")
		return strings.Contains(p, strings.Trim(g, "/"))
	}
	return strings.Contains(p, g) || filepath.Base(p) == g
}
