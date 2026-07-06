package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/control-plane/internal/store"
	"era/services/platform/licensegate"
)

func TestPatchPlanFromInventory(t *testing.T) {
	st := store.NewMemory()
	st.ReplaceAssetSoftware("n1", "t1", []*store.AssetSoftware{
		{NodeID: "n1", Name: "OpenSSL 3.0.13", Version: "3.0.13"},
	})
	srv := New(st, licensegate.DevAllEnabled())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/manage/patch/plan", nil)
	req.Header.Set("X-ERA-Role", "admin")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Plan []store.PatchPlanRow `json:"plan"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Plan) == 0 {
		t.Fatal("expected patch plan rows")
	}
}

func TestDeployJobCreate(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled())
	body := `{"node_id":"n1","package_ref":"s3://era-packages/app.msi"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/manage/deploy/jobs", bytes.NewReader([]byte(body)))
	req.Header.Set("X-ERA-Role", "admin")
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("deploy: %d %s", rec.Code, rec.Body.String())
	}
	var job store.DeployJob
	_ = json.Unmarshal(rec.Body.Bytes(), &job)
	rec2 := httptest.NewRecorder()
	patch := `{"status":"failed"}`
	req2 := httptest.NewRequest(http.MethodPatch, "/api/v1/manage/deploy/jobs/"+job.ID, bytes.NewReader([]byte(patch)))
	req2.Header.Set("X-ERA-Role", "admin")
	srv.Routes().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("rollback: %d %s", rec2.Code, rec2.Body.String())
	}
}
