package taxii

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/national-hub/internal/hub"
	"era/services/platform/licensegate"
)

func TestOutboundExportPseudonymized(t *testing.T) {
	st := hub.NewStore()
	st.Publish(hub.DefaultCollection, "org-bank-a", "obj-1", []byte(cleanBundle))
	exp := NewOutbound(st, licensegate.FromModules([]licensegate.Module{licensegate.ModuleNational}), "test-salt")
	req := httptest.NewRequest(http.MethodGet, "/taxii2/api1/outbound/export", nil)
	rr := httptest.NewRecorder()
	exp.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("export: %d %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Bundle json.RawMessage `json:"bundle"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	body := string(resp.Bundle)
	if contains(body, "org-bank-a") {
		t.Fatal("raw org id leaked in export")
	}
	if !contains(body, "pseudonymized") {
		t.Fatal("missing pseudonymized label")
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
