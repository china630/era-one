package docs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/ui/docs"
)

func TestDocsSPAIndex(t *testing.T) {
	h := docs.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !contains(rec.Body.String(), "ERA Documents") {
		t.Fatal("missing title")
	}
}

func TestDocsSPAFallbackDocPath(t *testing.T) {
	h := docs.Handler()
	req := httptest.NewRequest(http.MethodGet, "/doc-smoke-1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !contains(rec.Body.String(), "ERA Documents") {
		t.Fatal("expected index.html fallback for doc path")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
