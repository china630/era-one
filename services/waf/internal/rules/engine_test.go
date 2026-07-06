package rules

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOWASPBlocks(t *testing.T) {
	e := NewOWASP()
	cases := []struct {
		name string
		path string
		want string
	}{
		{"sqli", "/search?q=%27+OR+1%3D1--", "era-waf-sqli"},
		{"xss", "/page?x=%3Cscript%3Ealert(1)%3C/script%3E", "era-waf-xss"},
		{"traversal", "/files?f=..%2F..%2F..%2Fetc%2Fpasswd", "era-waf-path-traversal"},
		{"cmdi", "/run?cmd=%3B+cat+%2Fetc%2Fpasswd", "era-waf-cmdi"},
		{"ssrf", "/fetch?url=http://169.254.169.254/", "era-waf-ssrf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com"+tc.path, nil)
			m, ok := e.Evaluate(req)
			if !ok {
				t.Fatal("expected block")
			}
			if m.RuleID != tc.want {
				t.Fatalf("got %s want %s", m.RuleID, tc.want)
			}
		})
	}
}

func TestBenignAllowed(t *testing.T) {
	e := NewOWASP()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	if _, ok := e.Evaluate(req); ok {
		t.Fatal("benign request blocked")
	}
}
