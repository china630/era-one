package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/ai-core/internal/api"
	"era/services/ai-core/internal/investigate"
	"era/services/platform/licensegate"
)

func TestInvestigateListAndGet(t *testing.T) {
	srv := api.New((*investigate.Client)(nil), licensegate.FromModules([]licensegate.Module{licensegate.ModuleControlAI}))
	res := investigate.BuildResult(investigate.Request{DetectionID: "det-1", NodeID: "node-1"}, []investigate.StoryStep{
		{Summary: "powershell.exe"},
	})
	srv.Decisions.Put(res)
	if res.InvestigationID == "" || res.Status != "completed" {
		t.Fatalf("store did not assign id/status: %+v", res)
	}

	listRec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/api/v1/investigate", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("list want 200 got %d %s", listRec.Code, listRec.Body.String())
	}
	var listBody struct {
		Investigations []investigate.Result `json:"investigations"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listBody); err != nil {
		t.Fatal(err)
	}
	if len(listBody.Investigations) != 1 || listBody.Investigations[0].InvestigationID != res.InvestigationID {
		t.Fatalf("list %+v", listBody.Investigations)
	}

	getRec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/api/v1/investigate/"+res.InvestigationID, nil))
	if getRec.Code != http.StatusOK {
		t.Fatalf("get want 200 got %d %s", getRec.Code, getRec.Body.String())
	}
	var got investigate.Result
	if err := json.NewDecoder(getRec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.InvestigationID != res.InvestigationID || got.Verdict != res.Verdict || got.Status != "completed" {
		t.Fatalf("got %+v", got)
	}

	miss := httptest.NewRecorder()
	srv.Routes().ServeHTTP(miss, httptest.NewRequest(http.MethodGet, "/api/v1/investigate/missing", nil))
	if miss.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", miss.Code)
	}
}

func TestInvestigateListDeniedWithoutAIModule(t *testing.T) {
	srv := api.New((*investigate.Client)(nil), licensegate.FromModules(nil))
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/investigate", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d", rec.Code)
	}
}
