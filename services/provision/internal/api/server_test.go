package api

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"era/services/platform/cpclient"
	"era/services/platform/licensegate"
	"era/services/provision/internal/store"
)

var update = flag.Bool("update", false, "update golden files")

func TestEnrollRegistersAsset(t *testing.T) {
	var registered bool
	cpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/assets/register" {
			registered = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"policy_version":"1"}`))
		}
	}))
	defer cpSrv.Close()

	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), cpclient.New(cpSrv.URL))
	body := `{"agent_id":"a1","node_id":"n-prov-1","hostname":"bare-metal-01","platform":"linux"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("enroll: %d %s", rec.Code, rec.Body.String())
	}
	if !registered {
		t.Fatal("expected CMDB register")
	}
}

func TestEnrollRequiresControlPlane(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	body := `{"agent_id":"a1","node_id":"n1","hostname":"h1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestPXEConfigGolden(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pxe/config", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pxe: %d", rec.Code)
	}
	gotPath := filepath.Join("testdata", "pxe_config.golden.json")
	if *update {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, rec.Body.Bytes(), "", "  "); err != nil {
			t.Fatal(err)
		}
		pretty.WriteByte('\n')
		if err := os.WriteFile(gotPath, pretty.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatal(err)
	}
	var gotObj, wantObj any
	if err := json.Unmarshal(rec.Body.Bytes(), &gotObj); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &wantObj); err != nil {
		t.Fatal(err)
	}
	gb, _ := json.Marshal(gotObj)
	wb, _ := json.Marshal(wantObj)
	if !bytes.Equal(gb, wb) {
		t.Fatalf("pxe golden mismatch\ngot:  %s\nwant: %s\n(run with -update)", gb, wb)
	}
}

func TestImagesListAndDetail(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("images: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/images/img-linux-22", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("image detail: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/images/missing", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing image: %d", rec.Code)
	}
}

func TestLicenseGateProvision(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.FromModules(nil), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestImageCreateDeleteAndPXEPut(t *testing.T) {
	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), nil)
	h := srv.Routes()

	body := `{"id":"img-lab-1","name":"Lab Ubuntu","platform":"linux","version":"24.04","minio_ref":"s3://era-provision/images/lab.iso","unattended_kind":"preseed"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", bytes.NewReader([]byte(body)))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create image: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/images/img-lab-1", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get created: %d", rec.Code)
	}

	pxeBody := `{"tftp_root":"/tmp/tftp","default_image":"img-lab-1","boot_menu":[{"label":"Lab","image_id":"img-lab-1","kernel":"vmlinuz"}]}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/pxe/config", bytes.NewReader([]byte(pxeBody)))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("pxe put: %d %s", rec.Code, rec.Body.String())
	}
	var pxe map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &pxe)
	if pxe["default_image"] != "img-lab-1" {
		t.Fatalf("pxe: %v", pxe)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/images/img-lab-1", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/images/img-lab-1", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rec.Code)
	}
}

func TestEnrollRecordsJob(t *testing.T) {
	cpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/assets/register" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"policy_version":"1"}`))
		}
	}))
	defer cpSrv.Close()

	st := store.NewMemory()
	srv := New(st, licensegate.DevAllEnabled(), cpclient.New(cpSrv.URL))
	body := `{"agent_id":"a2","node_id":"n-job-1","hostname":"bare-02","platform":"linux","image_id":"img-linux-22"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", bytes.NewReader([]byte(body)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("enroll: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/enroll/jobs", nil)
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("jobs: %d", rec.Code)
	}
	var out struct {
		Jobs []map[string]any `json:"jobs"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Jobs) != 1 || out.Jobs[0]["status"] != "enrolled" {
		t.Fatalf("jobs=%v", out.Jobs)
	}
}
