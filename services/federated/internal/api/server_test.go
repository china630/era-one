package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/federated/internal/hub"
	"era/services/platform/licensegate"
)

func TestFederatedLicenseGate(t *testing.T) {
	s := New(hub.New(1.0), licensegate.DevDefault())
	body := `{"zone_id":"z1","vector":[0.1],"sample_count":10}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/federated/submit", strings.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleSubmit(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestFederatedWithLicense(t *testing.T) {
	s := New(hub.New(1.0), licensegate.FromModules([]licensegate.Module{licensegate.ModuleFederated}))
	body := `{"zone_id":"z1","vector":[0.1,0.2],"sample_count":10}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/federated/submit", strings.NewReader(body))
	rr := httptest.NewRecorder()
	s.handleSubmit(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rr.Code, rr.Body.String())
	}
}
