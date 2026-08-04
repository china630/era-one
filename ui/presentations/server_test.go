package presentations_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/ui/presentations"
)

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestPresentationsSPAIndex(t *testing.T) {
	h := presentations.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "ERA Presentations") {
		t.Fatal("missing title")
	}
	for _, want := range []string{
		"newDeckBtn", "addSlideBtn", "dupSlideBtn", "boldTextBtn", "insertImageBtn",
		"saveBtn", "importBtn", "exportBtn", "filmstrip", "shareDlg", "redoBtn",
		"presentImage",
	} {
		if !contains(body, want) {
			t.Fatalf("missing control %q", want)
		}
	}
}

func TestPresentationsSPAFallbackDeckPath(t *testing.T) {
	h := presentations.Handler()
	req := httptest.NewRequest(http.MethodGet, "/deck-e2e-1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !contains(rec.Body.String(), "ERA Presentations") {
		t.Fatal("expected index fallback for deck path")
	}
}

func TestPresentationsSPAAppFeatures(t *testing.T) {
	h := presentations.Handler()
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"/api/v1/presentations",
		"drive_object_id",
		"method: 'PUT'",
		"/api/v1/presentations/import",
		"/export/",
		"'pptx'",
		"'odp'",
		"exportOdp",
		"era_token",
		"addSlide",
		"duplicateSlide",
		"toggleBoldFormat",
		"stepFont",
		"insertSlideImage",
		"openShareDeck",
		"printSetup",
		"redoEdit",
		"pickDriveImageObject",
		"buildPrintRoot",
	} {
		if !contains(body, want) {
			t.Fatalf("missing %q in app.js", want)
		}
	}
}
