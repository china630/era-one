package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/ngfw/internal/policy"
	"era/services/platform/licensegate"
)

func TestEvaluateAPI(t *testing.T) {
	srv := New(policy.Default(), nil, licensegate.DevAllEnabled())
	body, _ := json.Marshal(policy.Flow{SrcIP: "203.0.113.1", DstIP: "8.8.8.8", Protocol: "tcp", DstPort: 445})
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/ngfw/evaluate", bytes.NewReader(body)))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	dec := out["decision"].(map[string]any)
	if dec["allowed"] != false {
		t.Fatalf("%v", out)
	}
}

func TestLicenseDenied(t *testing.T) {
	srv := New(policy.Default(), nil, licensegate.FromModules(nil))
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ngfw/policies", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("%d", rec.Code)
	}
}

func TestHealthzApplyBackend(t *testing.T) {
	t.Setenv("ERA_NGFW_APPLY", "")
	srv := New(policy.Default(), nil, licensegate.DevAllEnabled())
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["apply_backend"] != "noop" {
		t.Fatalf("want noop backend, got %v", out["apply_backend"])
	}
	if out["apply_enabled"] != false {
		t.Fatalf("want apply_enabled=false, got %v", out)
	}
}

func TestApplyNoopAndHistory(t *testing.T) {
	t.Setenv("ERA_NGFW_APPLY", "")
	srv := New(policy.Default(), nil, licensegate.DevAllEnabled())
	mux := srv.Routes()

	body := `{"id":"deny-lab","dst_port":445,"action":"deny"}`
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/ngfw/apply", bytes.NewReader([]byte(body))))
	if rec.Code != 200 {
		t.Fatalf("apply: %d %s", rec.Code, rec.Body.String())
	}
	var applyOut map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &applyOut)
	if applyOut["applied"] != false || applyOut["backend"] != "noop" {
		t.Fatalf("%v", applyOut)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ngfw/apply/history", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var hist struct {
		History []map[string]any `json:"history"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &hist)
	if len(hist.History) != 1 {
		t.Fatalf("history: %+v", hist)
	}
	if hist.History[0]["applied"] != false || hist.History[0]["backend"] != "noop" {
		t.Fatalf("%v", hist.History[0])
	}
}

func TestPolicyByIDAndIndex(t *testing.T) {
	srv := New(policy.Default(), nil, licensegate.DevAllEnabled())
	mux := srv.Routes()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ngfw/policies/deny-external-smb", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var byID policy.Rule
	_ = json.Unmarshal(rec.Body.Bytes(), &byID)
	if byID.ID != "deny-external-smb" || byID.DstPort != 445 {
		t.Fatalf("%+v", byID)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ngfw/policies/0", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var byIdx policy.Rule
	_ = json.Unmarshal(rec.Body.Bytes(), &byIdx)
	if byIdx.ID == "" {
		t.Fatal("empty rule at index 0")
	}
}
