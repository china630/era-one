package taxii

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/national-hub/internal/hub"
	"era/services/national-hub/internal/signature"
	"era/services/national-hub/internal/stix"
	"era/services/platform/licensegate"
)

const cleanBundle = `{
	"type":"bundle","id":"bundle--1","spec_version":"2.1",
	"objects":[{
		"type":"indicator","id":"indicator--evil","spec_version":"2.1",
		"name":"AZ IOC","pattern":"[domain-name:value='evil.az']",
		"pattern_type":"stix","confidence":90
	}]
}`

func TestE2EExchangeTwoOrgs(t *testing.T) {
	st := hub.NewStore()
	srv := New(st, licensegate.FromModules([]licensegate.Module{licensegate.ModuleNational}))
	h := srv.RoutesWithObjects()

	// Org A publishes
	req := httptest.NewRequest(http.MethodPost, "/taxii2/api1/collections/era-national-threats/objects/", bytes.NewBufferString(cleanBundle))
	req.Header.Set("X-ERA-Org-ID", "org-bank-a")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("publish: %d %s", rr.Code, rr.Body.String())
	}

	st.Subscribe("org-bank-b", hub.DefaultCollection)

	// Org B polls
	req2 := httptest.NewRequest(http.MethodGet, "/taxii2/api1/collections/era-national-threats/objects/", nil)
	req2.Header.Set("X-ERA-Org-ID", "org-bank-b")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("poll: %d", rr2.Code)
	}
	var resp struct {
		Objects []hub.Object `json:"objects"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(resp.Objects))
	}

	var bundle stix.Bundle
	if err := json.Unmarshal([]byte(resp.Objects[0].RawJSON), &bundle); err != nil {
		t.Fatal(err)
	}
	m := signature.FromBundle(&bundle, "national-hub")
	if m.Match(`{"query":"evil.az"}`) == 0 {
		t.Fatal("subscriber should detect national IOC")
	}
}

func TestNationalLicenseGate(t *testing.T) {
	srv := New(hub.NewStore(), licensegate.DevDefault())
	req := httptest.NewRequest(http.MethodGet, "/taxii2/api1/collections/", nil)
	rr := httptest.NewRecorder()
	srv.RoutesWithObjects().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestPublishBlocksPII(t *testing.T) {
	srv := New(hub.NewStore(), licensegate.FromModules([]licensegate.Module{licensegate.ModuleNational}))
	bad := `{"type":"bundle","objects":[{"pattern":"user=alice@corp.az"}]}`
	req := httptest.NewRequest(http.MethodPost, "/taxii2/api1/collections/era-national-threats/objects/", bytes.NewBufferString(bad))
	rr := httptest.NewRecorder()
	srv.RoutesWithObjects().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}
