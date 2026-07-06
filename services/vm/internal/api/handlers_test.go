package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/vm/internal/models"
	"era/services/vm/internal/scanner"
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

func TestHandleScan_OK(t *testing.T) {
	eng := scanner.NewEngine(staticExecutor{}, []*models.Template{{
		ID: "tpl-1",
		Info: models.Info{
			Name:     "Exposed Git Repository",
			Severity: "high",
		},
	}}, 2)

	payload, _ := json.Marshal(ScanRequest{
		Targets:     []string{"https://example.com"},
		Concurrency: 10,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vm/scan", bytes.NewReader(payload))
	rr := httptest.NewRecorder()

	HandleScan(eng, nil).ServeHTTP(rr, req)

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

	HandleScan(eng, nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
