package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/vm/internal/scheduler"
	"era/services/vm/internal/scanner"
)

func TestScheduleRoutes(t *testing.T) {
	sched := scheduler.New()
	mux := SetupRoutes(scanner.NewEngine(nil, nil, 1), nil, sched)

	body := `{"name":"weekly","targets":["127.0.0.1"],"cron_expr":"@every 24h","concurrency":2}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scans/schedule", strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("post: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/scans/schedule", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: %d", rec.Code)
	}
	var resp struct {
		Schedules []map[string]any `json:"schedules"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || len(resp.Schedules) != 1 {
		t.Fatalf("list: %v", rec.Body.String())
	}
}
