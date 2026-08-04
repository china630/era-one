package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/vm/internal/models"
	"era/services/vm/internal/scanner"
	"era/services/vm/internal/scheduler"
)

type staticExecutor struct{}

func (staticExecutor) Execute(target string, tpl *models.Template) ([]models.Finding, error) {
	return []models.Finding{{
		TemplateID:        tpl.ID,
		Target:            target,
		Severity:          tpl.Info.Severity,
		VulnerabilityName: tpl.Info.Name,
		MatchedURL:        target + "/.git/config",
	}}, nil
}

func testEngine() *scanner.Engine {
	return scanner.NewEngine(staticExecutor{}, []*models.Template{{
		ID: "tpl-1",
		Info: models.Info{
			Name:     "Exposed Git Repository",
			Severity: "high",
		},
	}}, 2)
}

func TestHandleScan_OK(t *testing.T) {
	eng := testEngine()
	store := NewJobStore()

	payload, _ := json.Marshal(ScanRequest{
		Targets:     []string{"https://example.com"},
		Concurrency: 10,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vm/scan", bytes.NewReader(payload))
	rr := httptest.NewRecorder()

	HandleScan(eng, nil, store).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusOK)
	}

	var resp ScanResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("response status: got %q", resp.Status)
	}
	if resp.JobID == "" {
		t.Fatal("expected job_id")
	}
	if resp.Mode != "http" {
		t.Fatalf("mode: got %q", resp.Mode)
	}
	if len(resp.Findings) != 1 {
		t.Fatalf("findings len: got %d, want 1", len(resp.Findings))
	}
	if resp.Findings[0].TemplateID != "tpl-1" {
		t.Fatalf("template id: got %q", resp.Findings[0].TemplateID)
	}
}

func TestHandleScan_InvalidJSON(t *testing.T) {
	eng := scanner.NewEngine(staticExecutor{}, nil, 1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vm/scan", bytes.NewBufferString("{bad json"))
	rr := httptest.NewRecorder()

	HandleScan(eng, nil, NewJobStore()).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestJobsAndFindingsRoutes(t *testing.T) {
	store := NewJobStore()
	mux := SetupRoutesWithJobs(testEngine(), nil, nil, store)

	payload := `{"targets":["https://example.com"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vm/scan", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("scan: %d %s", rec.Code, rec.Body.String())
	}
	var scan ScanResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &scan)

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vm/jobs", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("jobs: %d", rec.Code)
	}
	var list struct {
		Jobs []ScanJob `json:"jobs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list.Jobs) != 1 {
		t.Fatalf("list jobs: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vm/jobs/"+scan.JobID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("get job: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vm/findings", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("findings: %d", rec.Code)
	}
	var findings struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &findings)
	if findings.Count != 1 {
		t.Fatalf("findings count: %d", findings.Count)
	}
}

func TestCredentialedStubLabeled(t *testing.T) {
	store := NewJobStore()
	mux := SetupRoutesWithJobs(testEngine(), nil, nil, store)

	payload := `{"targets":["127.0.0.1:1"],"mode":"credentialed"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vm/scan", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("scan: %d %s", rec.Code, rec.Body.String())
	}
	var resp ScanResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Mode != "credentialed_stub" {
		t.Fatalf("mode: %q", resp.Mode)
	}
	if !strings.Contains(resp.Note, "credentialed_stub") {
		t.Fatalf("expected honest stub note, got %q", resp.Note)
	}
}

func TestSchedulePatchDelete(t *testing.T) {
	sched := scheduler.New()
	mux := SetupRoutes(testEngine(), nil, sched)

	body := `{"name":"weekly","targets":["127.0.0.1"],"cron_expr":"@every 24h","concurrency":2}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scans/schedule", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("post: %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Job map[string]any `json:"job"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id, _ := created.Job["id"].(string)
	if id == "" {
		t.Fatal("missing schedule id")
	}

	patch := `{"name":"nightly","enabled":false}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/scans/schedule/"+id, strings.NewReader(patch))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/scans/schedule/"+id, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	if len(sched.List()) != 0 {
		t.Fatal("expected empty schedules after delete")
	}
}
