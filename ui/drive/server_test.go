package drive_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/ui/drive"
)

func TestDriveSPAIndex(t *testing.T) {
	h := drive.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "ERA Drive") && !contains(body, "ERA Office") {
		t.Fatal("missing Drive branding")
	}
	if contains(body, "Dev login") || contains(body, "loginBtn") {
		t.Fatal("dev login panel must be removed — use /login")
	}
	if !contains(body, "userChip") {
		t.Fatal("missing account chip")
	}
}

func TestDriveSPAAppAuthRedirect(t *testing.T) {
	h := drive.Handler()
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if contains(body, "/oauth2/staging/token") {
		t.Fatal("Drive app.js must not call staging token directly — /login owns auth")
	}
	if !contains(body, "/login") {
		t.Fatal("Drive must redirect to /login when unsigned")
	}
	if !contains(body, "era_token") {
		t.Fatal("expected era_token usage")
	}
	if !contains(body, "createFolderBtn") && !contains(body, "/folders") {
		t.Fatal("expected create folder API usage")
	}
	if !contains(body, "breadcrumb") && !contains(body, "pathStack") {
		t.Fatal("expected folder navigation")
	}
	if !contains(body, "versions") {
		t.Fatal("expected versions UI")
	}
	if !contains(body, "newDocBtn") || !contains(body, "/api/v1/docs") {
		t.Fatal("expected New document → docs create")
	}
	if !contains(body, "Open in Docs") {
		t.Fatal("expected Open in Docs for .erad/.docx")
	}
	if !contains(body, "newSheetBtn") || !contains(body, "/api/v1/tables") {
		t.Fatal("expected New sheet → tables create")
	}
	if !contains(body, "Open in Tables") {
		t.Fatal("expected Open in Tables for .erat/.xlsx")
	}
	if !contains(body, "newDeckBtn") || !contains(body, "/api/v1/presentations") {
		t.Fatal("expected New presentation → presentations create")
	}
	if !contains(body, "Open in Presentations") {
		t.Fatal("expected Open in Presentations for .erap/.pptx")
	}
	if !contains(body, "newProjectBtn") || !contains(body, "/api/v1/projects") {
		t.Fatal("expected New project → projects create (.eraj)")
	}
	if !contains(body, "Open in Projects") {
		t.Fatal("expected Open in Projects for .eraj")
	}
}

func TestDriveSPAHasFolderControls(t *testing.T) {
	h := drive.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{
		"createFolderBtn", "breadcrumb", "versionsPanel", "Upload",
		"newDocBtn", "newSheetBtn", "newDeckBtn", "newProjectBtn",
	} {
		if !contains(body, want) {
			t.Fatalf("missing %q in index", want)
		}
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
