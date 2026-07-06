package parser

import (
	"strings"
	"testing"
)

// TestParseTemplate_ExposedGitConfig проверяет разбор упрощённого шаблона в духе Nuclei (.git/config).
func TestParseTemplate_ExposedGitConfig(t *testing.T) {
	const yamlDoc = `
id: exposed-git-config
info:
  name: Exposed Git Repository
  author: vm-mvp
  severity: high
  description: Обнаружен публичный доступ к /.git/config
requests:
  - method: GET
    path:
      - /.git/config
    headers:
      User-Agent: era-vm/0.1
    matchers:
      - type: word
        part: body
        words:
          - "[core]"
        condition: and
`

	tmpl, err := ParseTemplate([]byte(yamlDoc))
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}

	if tmpl.ID != "exposed-git-config" {
		t.Errorf("ID: got %q, want exposed-git-config", tmpl.ID)
	}
	if tmpl.Info.Name != "Exposed Git Repository" {
		t.Errorf("Info.Name: got %q", tmpl.Info.Name)
	}
	if tmpl.Info.Author != "vm-mvp" {
		t.Errorf("Info.Author: got %q", tmpl.Info.Author)
	}
	if tmpl.Info.Severity != "high" {
		t.Errorf("Info.Severity: got %q", tmpl.Info.Severity)
	}
	if !strings.Contains(tmpl.Info.Description, ".git/config") {
		t.Errorf("Info.Description: got %q", tmpl.Info.Description)
	}

	if len(tmpl.Requests) != 1 {
		t.Fatalf("Requests len: got %d, want 1", len(tmpl.Requests))
	}
	req := tmpl.Requests[0]
	if req.Method != "GET" {
		t.Errorf("Method: got %q", req.Method)
	}
	if len(req.Path) != 1 || req.Path[0] != "/.git/config" {
		t.Errorf("Path: got %#v", req.Path)
	}
	if got := req.Headers["User-Agent"]; got != "era-vm/0.1" {
		t.Errorf("Headers[User-Agent]: got %q", got)
	}
	if len(req.Matchers) != 1 {
		t.Fatalf("Matchers len: got %d", len(req.Matchers))
	}
	m := req.Matchers[0]
	if m.Type != "word" || m.Part != "body" {
		t.Errorf("Matcher type/part: got %q / %q", m.Type, m.Part)
	}
	if len(m.Words) != 1 || m.Words[0] != "[core]" {
		t.Errorf("Matcher.Words: got %#v", m.Words)
	}
	if m.Condition != "and" {
		t.Errorf("Matcher.Condition: got %q", m.Condition)
	}
}

func TestParseTemplate_InvalidFormat(t *testing.T) {
	_, err := ParseTemplate([]byte(`id: ""
requests: []
`))
	if err == nil {
		t.Fatal("ожидалась ошибка валидации")
	}
	if err.Error() != "invalid template format" {
		t.Fatalf("ошибка: got %q, want invalid template format", err.Error())
	}
}