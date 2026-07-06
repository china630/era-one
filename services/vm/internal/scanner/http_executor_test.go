package scanner

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"era/services/vm/internal/models"
)

func TestMatchResponse_WordBody_And(t *testing.T) {
	body := []byte(`[core] repositoryformatversion = 0`)
	ok := matchResponse(200, nil, body, []models.Matcher{
		{Type: "word", Part: "body", Words: []string{"[core]", "repositoryformatversion"}, Condition: "and"},
	})
	if !ok {
		t.Fatal("ожидалось совпадение AND по словам в body")
	}
}

func TestMatchResponse_WordBody_Or(t *testing.T) {
	body := []byte("alpha only")
	ok := matchResponse(200, nil, body, []models.Matcher{
		{Type: "word", Part: "body", Words: []string{"missing", "alpha"}, Condition: "or"},
	})
	if !ok {
		t.Fatal("ожидалось совпадение OR")
	}
}

func TestMatchResponse_Status(t *testing.T) {
	if !matchResponse(403, nil, nil, []models.Matcher{{Type: "status", Status: []int{200, 403, 404}}}) {
		t.Fatal("ожидалось совпадение по статусу 403")
	}
	if matchResponse(500, nil, nil, []models.Matcher{{Type: "status", Status: []int{200}}}) {
		t.Fatal("не должно совпадать при другом статусе")
	}
}

func TestMatchResponse_MultipleMatchers_AND(t *testing.T) {
	body := []byte("secret-token")
	h := http.Header{}
	h.Set("X-App", "legacy")
	ok := matchResponse(200, h, body, []models.Matcher{
		{Type: "word", Part: "body", Words: []string{"secret-token"}},
		{Type: "word", Part: "header", Words: []string{"legacy"}},
	})
	if !ok {
		t.Fatal("ожидалось совпадение обоих матчеров")
	}
}

func TestHTTPExecutor_NoRedirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/a" {
			http.Redirect(w, r, "/b", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	ex := NewHTTPExecutor()
	tpl := &models.Template{
		ID: "t1",
		Info: models.Info{
			Name:     "redirect probe",
			Severity: "info",
		},
		Requests: []models.Request{
			{
				Method: http.MethodGet,
				Path:   []string{"/a"},
				Matchers: []models.Matcher{
					{Type: "status", Status: []int{302}},
				},
			},
		},
	}

	findings, err := ex.Execute(ts.URL, tpl)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings: got %d, want 1", len(findings))
	}
	if findings[0].MatchedURL == "" {
		t.Fatal("ожидался MatchedURL")
	}
}

func TestHTTPExecutor_WordMatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello [core] world"))
	}))
	defer ts.Close()

	ex := NewHTTPExecutor()
	tpl := &models.Template{
		ID: "git-like",
		Info: models.Info{
			Name:     "Exposed Git",
			Severity: "high",
		},
		Requests: []models.Request{
			{
				Method: http.MethodGet,
				Path:   []string{"/"},
				Matchers: []models.Matcher{
					{Type: "word", Part: "body", Words: []string{"[core]"}},
				},
			},
		},
	}

	findings, err := ex.Execute(ts.URL, tpl)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings: got %d", len(findings))
	}
	f := findings[0]
	if f.TemplateID != "git-like" || f.VulnerabilityName != "Exposed Git" {
		t.Fatalf("finding fields: %+v", f)
	}
	if f.Timestamp.IsZero() || f.Timestamp.After(time.Now().Add(time.Minute)) {
		t.Fatalf("unexpected timestamp: %v", f.Timestamp)
	}
}
