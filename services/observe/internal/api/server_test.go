package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/observe/internal/cmdb"
	ingestclient "era/services/observe/internal/ingest"
	"era/services/platform/licensegate"
)

func TestPRTGWebhookAccepts(t *testing.T) {
	ing := ingestclient.New("", "t1")
	srv := New(ing, cmdb.New(""), licensegate.DevAllEnabled(), "t1")
	body := []byte(`{"host":"10.0.0.1","message":"high egress on uplink","sensor":"Traffic"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/prtg", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out["node_id"] != "net-10-0-0-1" {
		t.Fatalf("%v", out)
	}
}

func TestObserveLicenseGate(t *testing.T) {
	gate := licensegate.FromModules(nil)
	srv := New(ingestclient.New("", "t1"), cmdb.New(""), gate, "t1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/prtg", bytes.NewReader([]byte(`{"host":"x"}`)))
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", rr.Code)
	}
}

func TestTopologyWidget(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"assets": []cmdb.NetworkAsset{
				{NodeID: "sw-01", Hostname: "core-sw", AssetKind: "switch", Managed: true},
			},
		})
	}))
	defer mock.Close()
	srv := New(ingestclient.New("", "t1"), cmdb.New(mock.URL), licensegate.DevAllEnabled(), "t1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topology", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var out struct {
		Nodes []map[string]string `json:"nodes"`
		Edges []map[string]string `json:"edges"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if len(out.Nodes) != 1 || len(out.Edges) != 1 {
		t.Fatalf("topology=%+v", out)
	}
}
