package projects_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/ui/projects"
)

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestProjectsSPAIndex(t *testing.T) {
	h := projects.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "ERA Projects") {
		t.Fatal("missing title")
	}
	for _, want := range []string{
		"addBtn", "refreshBtn", "board", "taskTitle", "driveObjectId",
		"shareDlg", "taskPriority", "filterPriority", "editPriority",
	} {
		if !contains(body, want) {
			t.Fatalf("missing control %q", want)
		}
	}
}

func TestProjectsSPAAppFeatures(t *testing.T) {
	h := projects.Handler()
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"/api/v1/projects/tasks",
		"era_token",
		"drive_object_id",
		"Open in Docs",
		"backlog",
		"DELETE",
		"moveTask",
		"openShareProject",
		"normalizePriority",
		"moveTaskFull",
		"viewModeStorageKey",
		"loadDrivePickerList",
	} {
		if !contains(body, want) {
			t.Fatalf("missing %q in app.js", want)
		}
	}
}
