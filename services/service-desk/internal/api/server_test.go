package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"era/services/platform/cpclient"
	"era/services/platform/licensegate"
	"era/services/service-desk/internal/store"
)
func TestIncidentLifecycle(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)

	body := `{"title":"VPN down","node_id":"n1","requester":"user1","sla_hours":4}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var inc map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &inc)
	id, _ := inc["id"].(string)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/incidents/"+id, bytes.NewReader([]byte(`{"status":"in_progress","assignee":"tech1"}`)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d", rec.Code)
	}
}

func TestServiceRequestPortal(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	body := `{"title":"New laptop","requester":"user2","category":"hardware"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/requests", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("request: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/requests", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list requests: %d", rec.Code)
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	reqs, _ := out["requests"].([]any)
	if len(reqs) != 1 {
		t.Fatalf("want 1 request, got %d", len(reqs))
	}
}

func TestProblemCreateList(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	body := `{"title":"Recurring VPN flap","node_id":"n1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/problems", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("problem: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/problems", nil)
	srv.Routes().ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out["problems"].([]any)) != 1 {
		t.Fatal("problems list")
	}
}

func TestChangeCreateList(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	body := `{"title":"Firewall rule","risk":"medium"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/changes", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("change: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/changes", nil)
	srv.Routes().ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out["changes"].([]any)) != 1 {
		t.Fatal("changes list")
	}
}

func TestIncidentNodeNotInCMDB(t *testing.T) {
	cpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer cpSrv.Close()
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), cpclient.New(cpSrv.URL))
	body := `{"title":"bad node","node_id":"missing","requester":"u1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestLicenseGateService(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.FromModules(nil), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestUIServesIndex(t *testing.T) {
	root := filepath.Join("..", "..", "..", "ui", "service-desk")
	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		t.Skip("ui/service-desk not found from test cwd")
	}
	t.Setenv("ERA_UI_DIR", root)
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusMovedPermanently {
		// FileServer may redirect or serve
		rec2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/ui/index.html", nil)
		srv.Routes().ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusOK {
			t.Fatalf("ui: %d / index %d", rec.Code, rec2.Code)
		}
	}
}

func TestTicketDetailPatchAndComments(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	h := srv.Routes()

	post := func(path, body string) map[string]any {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(body)))
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("POST %s: %d %s", path, rec.Code, rec.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return out
	}
	patch := func(path, body string) map[string]any {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader([]byte(body)))
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("PATCH %s: %d %s", path, rec.Code, rec.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return out
	}
	get := func(path string) map[string]any {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", path, rec.Code, rec.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return out
	}

	inc := post("/api/v1/incidents", `{"title":"VPN","requester":"u1","sla_hours":4}`)
	incID, _ := inc["id"].(string)
	if inc["sla_status"] != "ok" {
		t.Fatalf("incident sla_status=%v", inc["sla_status"])
	}
	got := get("/api/v1/incidents/" + incID)
	if got["id"] != incID || got["sla_status"] != "ok" {
		t.Fatalf("incident detail: %v", got)
	}
	cmt := post("/api/v1/incidents/"+incID+"/comments", `{"author":"tech1","body":"looking"}`)
	if cmt["body"] != "looking" {
		t.Fatalf("comment: %v", cmt)
	}
	clist := get("/api/v1/incidents/" + incID + "/comments")
	if len(clist["comments"].([]any)) != 1 {
		t.Fatal("comments list")
	}

	req := post("/api/v1/requests", `{"title":"Laptop","requester":"u2"}`)
	reqID, _ := req["id"].(string)
	patched := patch("/api/v1/requests/"+reqID, `{"status":"in_progress","assignee":"tech2"}`)
	if patched["status"] != "in_progress" || patched["assignee"] != "tech2" {
		t.Fatalf("request patch: %v", patched)
	}
	if get("/api/v1/requests/"+reqID)["status"] != "in_progress" {
		t.Fatal("request detail")
	}
	_ = post("/api/v1/requests/"+reqID+"/comments", `{"author":"u2","body":"urgent"}`)

	prob := post("/api/v1/problems", `{"title":"Recurring flap"}`)
	probID, _ := prob["id"].(string)
	patch("/api/v1/problems/"+probID, `{"status":"in_progress"}`)
	if get("/api/v1/problems/"+probID)["status"] != "in_progress" {
		t.Fatal("problem detail")
	}

	chg := post("/api/v1/changes", `{"title":"FW rule","risk":"low"}`)
	chgID, _ := chg["id"].(string)
	patch("/api/v1/changes/"+chgID, `{"status":"in_progress","risk":"medium"}`)
	detail := get("/api/v1/changes/" + chgID)
	if detail["risk"] != "medium" {
		t.Fatalf("change detail: %v", detail)
	}
}
