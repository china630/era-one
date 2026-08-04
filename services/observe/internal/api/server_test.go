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

func TestDeviceDetailAlertsPollers(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"assets": []cmdb.NetworkAsset{
				{NodeID: "sw-01", Hostname: "core-sw", AssetKind: "switch", Managed: true, IPAddrs: []string{"10.0.0.1"}},
			},
		})
	}))
	defer mock.Close()
	srv := New(ingestclient.New("", "t1"), cmdb.New(mock.URL), licensegate.DevAllEnabled(), "t1")
	h := srv.Routes()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/devices/sw-01", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("device detail: %d %s", rr.Code, rr.Body.String())
	}
	var detail map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &detail)
	if detail["metrics_source"] != "sim" {
		t.Fatalf("want metrics_source=sim, got %v", detail["metrics_source"])
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/devices/missing", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing device: %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("alerts: %d", rr.Code)
	}
	var alertsWrap struct {
		Alerts []map[string]any `json:"alerts"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &alertsWrap)
	if len(alertsWrap.Alerts) < 1 {
		t.Fatal("expected lab alerts")
	}
	alertID, _ := alertsWrap.Alerts[0]["id"].(string)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/alerts/"+alertID+"/ack", bytes.NewReader([]byte(`{"by":"ops"}`))))
	if rr.Code != http.StatusOK {
		t.Fatalf("ack: %d %s", rr.Code, rr.Body.String())
	}
	var acked map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &acked)
	if acked["acked"] != true || acked["acked_by"] != "ops" {
		t.Fatalf("acked: %v", acked)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/pollers", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("pollers list: %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/pollers", bytes.NewReader([]byte(
		`{"name":"edge","targets":["10.0.0.2"],"interval_sec":30,"enabled":true}`,
	))))
	if rr.Code != http.StatusCreated {
		t.Fatalf("poller create: %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &created)
	pid, _ := created["id"].(string)

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/pollers/"+pid, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("poller get: %d", rr.Code)
	}
}
