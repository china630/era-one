package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestProjectsWithoutJWT401(t *testing.T) {
	s := &server{
		store:     newMemStore(),
		jwtSecret: []byte("test-secret"),
		licenseOK: true,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/tasks", nil)
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestProjectsWithoutLicense403(t *testing.T) {
	s := &server{
		store:     newMemStore(),
		jwtSecret: []byte("test-secret"),
		licenseOK: false,
	}
	tok := mustJWT(t, "test-secret", "t1", "u1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestProjectsCreateWithDeepLink(t *testing.T) {
	s := &server{
		store:     newMemStore(),
		jwtSecret: []byte("test-secret"),
		licenseOK: true,
	}
	tok := mustJWT(t, "test-secret", "t1", "u1")
	body := `{"title":"Write brief","board":"backlog","drive_object_id":"drv-obj-1","assignee":"alice","due_date":"2026-08-01","labels":["design","p0"],"checklist":[{"id":"c1","text":"Draft outline","done":false},{"id":"c2","text":"Review","done":true}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/tasks", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got task
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.DriveObjectID != "drv-obj-1" || got.TenantID != "t1" || got.ID == "" {
		t.Fatalf("unexpected task: %+v", got)
	}
	if got.Assignee != "alice" || got.DueDate != "2026-08-01" {
		t.Fatalf("want assignee/due, got %+v", got)
	}
	if len(got.Labels) != 2 || got.Labels[0] != "design" || got.Labels[1] != "p0" {
		t.Fatalf("want labels, got %+v", got.Labels)
	}
	if len(got.Checklist) != 2 || got.Checklist[0].ID != "c1" || got.Checklist[0].Done || !got.Checklist[1].Done {
		t.Fatalf("want checklist, got %+v", got.Checklist)
	}
}

func TestProjectsPriorityAndSortKey(t *testing.T) {
	s := &server{
		store:     newMemStore(),
		jwtSecret: []byte("test-secret"),
		licenseOK: true,
	}
	tok := mustJWT(t, "test-secret", "t1", "u1")
	body := `{"title":"Urgent","board":"todo","priority":"p0","sort_key":10,"assignee":"bob"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/tasks", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got task
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Priority != "p0" || got.SortKey != 10 || got.Assignee != "bob" {
		t.Fatalf("want priority=p0 sort_key=10 assignee=bob, got %+v", got)
	}
	// Invalid priority cleared.
	body2 := `{"id":"` + got.ID + `","title":"Urgent","board":"todo","priority":"critical","sort_key":11}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/projects/tasks", strings.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer "+tok)
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr2, req2)
	var got2 task
	_ = json.NewDecoder(rr2.Body).Decode(&got2)
	if got2.Priority != "" {
		t.Fatalf("want cleared priority, got %q", got2.Priority)
	}
}

func TestProjectsBoardRename(t *testing.T) {
	s := &server{
		store:     newMemStore(),
		jwtSecret: []byte("test-secret"),
		licenseOK: true,
	}
	tok := mustJWT(t, "test-secret", "t1", "u1")
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/board", strings.NewReader(`{"name":"Sprint 12"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/projects/board", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	rr2 := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr2, req2)
	var b boardMeta
	if err := json.NewDecoder(rr2.Body).Decode(&b); err != nil {
		t.Fatal(err)
	}
	if b.Name != "Sprint 12" {
		t.Fatalf("got %q", b.Name)
	}
}

func TestProjectsDeleteTask(t *testing.T) {
	s := &server{
		store:     newMemStore(),
		jwtSecret: []byte("test-secret"),
		licenseOK: true,
	}
	tok := mustJWT(t, "test-secret", "t1", "u1")
	_ = s.store.put(nil, task{ID: "task-del", Title: "X", Board: "todo", TenantID: "t1"})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/tasks/task-del", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	newMux(s).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rr.Code)
	}
	list, _ := s.store.list(nil, "t1")
	if len(list) != 0 {
		t.Fatalf("expected empty after delete, got %+v", list)
	}
}

func TestLicenseFailClosedInProduction(t *testing.T) {
	t.Setenv("ERA_PRODUCTION", "1")
	t.Setenv("ERA_OFFICE_DEV", "")
	t.Setenv("ERA_LICENSE_OFFICE_PROJECTS", "")
	if licenseFromEnv() {
		t.Fatal("expected fail-closed without ERA_LICENSE_OFFICE_PROJECTS")
	}
}

func TestErajCreateFlushReopenSameID(t *testing.T) {
	drv := newMemErajDrive()
	s := &server{
		store:     newMemStore(),
		jwtSecret: []byte("test-secret"),
		licenseOK: true,
		drive:     drv,
		eraj:      newErajSessionCache(),
	}
	tok := mustJWT(t, "test-secret", "t1", "u1")
	mux := newMux(s)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(`{"name":"Sprint.eraj"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var created struct {
		DriveObjectID string `json:"drive_object_id"`
		Name          string `json:"name"`
		Format        string `json:"format"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.DriveObjectID == "" || created.Format != "eraj" || !strings.HasSuffix(created.Name, ".eraj") {
		t.Fatalf("unexpected create: %+v", created)
	}

	taskBody := `{"title":"Ship eraj","board":"doing","labels":["p0"],"checklist":[]}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+created.DriveObjectID+"/tasks", strings.NewReader(taskBody))
	req2.Header.Set("Authorization", "Bearer "+tok)
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("task want 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}

	// Drop session cache — reopen must load from Drive blob.
	s.eraj = newErajSessionCache()
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+created.DriveObjectID, nil)
	req3.Header.Set("Authorization", "Bearer "+tok)
	rr3 := httptest.NewRecorder()
	mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("reopen want 200, got %d", rr3.Code)
	}
	var doc ErajProject
	if err := json.NewDecoder(rr3.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc.DriveObjectID != created.DriveObjectID {
		t.Fatalf("id changed: %q vs %q", doc.DriveObjectID, created.DriveObjectID)
	}
	if len(doc.Tasks) != 1 || doc.Tasks[0].Title != "Ship eraj" || doc.Tasks[0].Board != "doing" {
		t.Fatalf("tasks not persisted: %+v", doc.Tasks)
	}
}

func TestUniqueErajNames(t *testing.T) {
	a := uniqueErajName("Board")
	b := uniqueErajName("Board")
	if a == b || !strings.HasSuffix(a, ".eraj") || !strings.HasSuffix(b, ".eraj") {
		t.Fatalf("want unique .eraj names, got %q %q", a, b)
	}
}

func mustJWT(t *testing.T, secret, tenant, sub string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"tenant_id": tenant,
		"sub":       sub,
	})
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}
