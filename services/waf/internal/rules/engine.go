// Package rules — WAF rule engine (Coraza-паттерн, реализация с нуля, F3-1).
package rules

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Match struct {
	RuleID   string
	Category string
	Severity string
}

type Rule struct {
	ID       string
	Category string
	Severity string
	re       *regexp.Regexp
	inspect  func(*http.Request) bool
}

type Engine struct {
	rules []Rule
}

func NewOWASP() *Engine {
	return &Engine{rules: owaspRules()}
}

func (e *Engine) Evaluate(r *http.Request) (Match, bool) {
	target := r.URL.Path
	if q, err := url.QueryUnescape(r.URL.RawQuery); err == nil {
		target += " " + q
	} else {
		target += " " + r.URL.RawQuery
	}
	for _, h := range []string{"User-Agent", "Cookie", "Referer"} {
		target += " " + r.Header.Get(h)
	}
	if r.Method == http.MethodPost {
		// body not read in MVP — path/query/header coverage for pen-test corpus
	}

	for _, rule := range e.rules {
		if rule.re != nil && rule.re.MatchString(target) {
			return Match{RuleID: rule.ID, Category: rule.Category, Severity: rule.Severity}, true
		}
		if rule.inspect != nil && rule.inspect(r) {
			return Match{RuleID: rule.ID, Category: rule.Category, Severity: rule.Severity}, true
		}
	}
	return Match{}, false
}

func owaspRules() []Rule {
	return []Rule{
		{ID: "era-waf-sqli", Category: "A03-injection", Severity: "critical",
			re: regexp.MustCompile(`(?i)('|\")(\s)*(or|union|select|drop|insert|delete)\s`)},
		{ID: "era-waf-xss", Category: "A03-injection", Severity: "high",
			re: regexp.MustCompile(`(?i)<\s*script|javascript:|onerror\s*=`)},
		{ID: "era-waf-path-traversal", Category: "A01-broken-access", Severity: "high",
			re: regexp.MustCompile(`(?i)(\.\./|\.\.\\|%2e%2e%2f)`)},
		{ID: "era-waf-cmdi", Category: "A03-injection", Severity: "critical",
			re: regexp.MustCompile(`(?i)(;\s*(cat|wget|curl|bash|sh)\s|&&\s*(cat|wget|curl))`)},
		{ID: "era-waf-ssrf", Category: "A10-ssrf", Severity: "medium",
			re: regexp.MustCompile(`(?i)(169\.254\.|127\.0\.0\.1|metadata\.google)`)},
		{ID: "era-waf-xxe", Category: "A05-misconfig", Severity: "high",
			inspect: func(r *http.Request) bool {
				return strings.Contains(r.Header.Get("Content-Type"), "xml") &&
					strings.Contains(r.URL.RawQuery, "ENTITY")
			}},
	}
}
