package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/platform/licensegate"
	"era/services/resolve/internal/atlas"
	"era/services/resolve/internal/guard"
	"era/services/resolve/internal/policy"
	"era/services/resolve/internal/trace"
)

func TestVerdictAndLicense(t *testing.T) {
	pol := policy.NewStore()
	atl := atlas.New()
	eng := guard.New(pol, atl)
	tr := trace.New(16, nil)
	srv := New(eng, pol, atl, tr, licensegate.DevAllEnabled())

	body, _ := json.Marshal(map[string]string{"qname": "a.malware.test", "qtype": "A"})
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/resolve/verdict", bytes.NewReader(body)))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var v map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &v)
	if v["action"] != "nxdomain" {
		t.Fatalf("%v", v)
	}

	denied := New(eng, pol, atl, tr, licensegate.FromModules(nil))
	rec = httptest.NewRecorder()
	denied.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/resolve/verdict", bytes.NewReader(body)))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("%d", rec.Code)
	}
}

func TestPolicyRuleCRUDAndTraceFilter(t *testing.T) {
	pol := policy.NewStore()
	pol.Replace(nil)
	atl := atlas.New()
	eng := guard.New(pol, atl)
	tr := trace.New(16, nil)
	srv := New(eng, pol, atl, tr, licensegate.DevAllEnabled())
	mux := srv.Routes()

	body := `{"id":"r1","suffix":".evil.lab","action":"nxdomain","reason":"test"}`
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/resolve/policy/rules", bytes.NewReader([]byte(body))))
	if rec.Code != http.StatusCreated {
		t.Fatalf("add: %d %s", rec.Code, rec.Body.String())
	}

	verdictBody, _ := json.Marshal(map[string]string{"qname": "x.evil.lab", "qtype": "A"})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/resolve/verdict", bytes.NewReader(verdictBody)))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	allowBody, _ := json.Marshal(map[string]string{"qname": "safe.example", "qtype": "A"})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/resolve/verdict", bytes.NewReader(allowBody)))

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/resolve/trace?q=evil", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var traceOut struct {
		Recent []trace.Record `json:"recent"`
		Q      string         `json:"q"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &traceOut)
	if traceOut.Q != "evil" || len(traceOut.Recent) != 1 {
		t.Fatalf("filter: %+v", traceOut)
	}
	if traceOut.Recent[0].Verdict.QName != "x.evil.lab" {
		t.Fatalf("qname: %s", traceOut.Recent[0].Verdict.QName)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/resolve/policy/rules/r1", nil))
	if rec.Code != 200 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	if _, ok := pol.Get("r1"); ok {
		t.Fatal("rule still present")
	}
}

func TestPackDeleteByID(t *testing.T) {
	pol := policy.NewStore()
	atl := atlas.New()
	_ = atl.Load(atlas.Pack{ID: "lab-v1", Version: "1", Domains: []atlas.Entry{{Domain: "bad.example", Severity: "high"}}})
	eng := guard.New(pol, atl)
	tr := trace.New(8, nil)
	srv := New(eng, pol, atl, tr, licensegate.DevAllEnabled())
	mux := srv.Routes()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/resolve/packs/wrong", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/resolve/packs/lab-v1", nil))
	if rec.Code != 200 {
		t.Fatalf("delete pack: %d %s", rec.Code, rec.Body.String())
	}
	if atl.Meta().ID != "" {
		t.Fatalf("pack not cleared: %+v", atl.Meta())
	}
}
