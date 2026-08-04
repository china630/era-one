package tables_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/ui/tables"
)

func TestTablesSPAIndex(t *testing.T) {
	h := tables.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "ERA Tables") {
		t.Fatal("missing title")
	}
	for _, want := range []string{
		"newSheetBtn", "sumBtn", "avgBtn", "roundBtn", "cellBoldBtn",
		"pasteValuesBtn", "importBtn", "exportBtn", "formulaInput",
	} {
		if !contains(body, want) {
			t.Fatalf("missing control %q", want)
		}
	}
}

func TestTablesSPAFallbackSheetPath(t *testing.T) {
	h := tables.Handler()
	req := httptest.NewRequest(http.MethodGet, "/sheet-e2e-1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !contains(rec.Body.String(), "ERA Tables") {
		t.Fatal("expected index.html fallback for sheet path")
	}
}

func TestTablesSPAAppFeatures(t *testing.T) {
	h := tables.Handler()
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"/api/v1/tables",
		"drive_object_id",
		"set_cell",
		"=SUM",
		"=AVERAGE",
		"set_cell_style",
		"pasteValuesActive",
		"/api/v1/tables/import",
		"export/xlsx",
		"ArrowDown",
		"era_token",
	} {
		if !contains(body, want) {
			t.Fatalf("missing %q in app.js", want)
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
