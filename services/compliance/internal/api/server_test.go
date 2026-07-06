package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/platform/licensegate"
)

func TestRegulatoryRequiresNationalLicense(t *testing.T) {
	s := New(licensegate.DevDefault())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/regulatory", nil)
	rr := httptest.NewRecorder()
	s.handleRegulatory(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestRegulatoryOK(t *testing.T) {
	s := New(licensegate.FromModules([]licensegate.Module{licensegate.ModuleNational}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/regulatory?org=DemoBank", nil)
	rr := httptest.NewRecorder()
	s.handleRegulatory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d %s", rr.Code, rr.Body.String())
	}
}
