package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/platform/licensegate"
	"era/services/soar/internal/api"
	"era/services/soar/internal/playbooks"
)

func TestPlaybooksCatalog(t *testing.T) {
	srv := api.New(playbooks.NewFromEnv(), licensegate.FromModules([]licensegate.Module{licensegate.ModuleResponse}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/playbooks", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Playbooks []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"playbooks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"isolate_host": true, "block_ip": true, "create_ticket": true}
	if len(body.Playbooks) != 3 {
		t.Fatalf("want 3 playbooks got %d", len(body.Playbooks))
	}
	for _, p := range body.Playbooks {
		if !want[p.Name] || p.Description == "" {
			t.Fatalf("unexpected entry %+v", p)
		}
	}
}

func TestActionGetByID(t *testing.T) {
	eng := playbooks.NewFromEnv()
	a := eng.IsolateHost("node-lab")
	srv := api.New(eng, licensegate.FromModules([]licensegate.Module{licensegate.ModuleResponse}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/actions/"+a.ID, nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d %s", rec.Code, rec.Body.String())
	}
	var got playbooks.Action
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != a.ID || got.Playbook != "isolate_host" {
		t.Fatalf("got %+v want id=%s", got, a.ID)
	}

	miss := httptest.NewRecorder()
	srv.Routes().ServeHTTP(miss, httptest.NewRequest(http.MethodGet, "/api/v1/actions/missing-id", nil))
	if miss.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", miss.Code)
	}
}
