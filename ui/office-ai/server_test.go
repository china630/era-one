package officeai_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	officeai "era/ui/office-ai"
)

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestOfficeAISPAIndex(t *testing.T) {
	h := officeai.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "ERA Office AI") {
		t.Fatal("missing title")
	}
	for _, want := range []string{"summarizeBtn", "sourceText", "modeBadge", "Air-gap"} {
		if !contains(body, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestOfficeAISPAApp(t *testing.T) {
	h := officeai.Handler()
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"/api/v1/docs-ai/summarize",
		"era_token",
		"mode",
		"application/json",
	} {
		if !contains(body, want) {
			t.Fatalf("missing %q in app.js", want)
		}
	}
}
