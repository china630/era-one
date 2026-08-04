package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/comms/mail-bridge/internal/audit"
	"era/services/comms/mail-bridge/internal/upstream"
	"era/services/platform/licensegate"
)

func TestHealthzAndAutodiscoverGolden(t *testing.T) {
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsOutlookBridge})
	stub := upstream.StubBackend{}
	router, _ := upstream.LoadFromEnv(stub)
	srv := NewServer(gate, router, &audit.Recorder{})
	srv.UseTLS = true
	mux := http.NewServeMux()
	srv.Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"upstream_mode":"stub"`) {
		t.Fatalf("want upstream_mode=stub in healthz: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/autodiscover/autodiscover.xml?email=alice@mail.gov.az", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("autodiscover status %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<SSL>on</SSL>") {
		t.Fatalf("expected SSL on in autodiscover: %s", body)
	}
	if !strings.Contains(body, "/ews/Exchange.asmx") {
		t.Fatalf("expected EwsUrl: %s", body)
	}
}

func TestEWSStub502(t *testing.T) {
	t.Setenv("ERA_BRIDGE_DEV", "1")
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsOutlookBridge})
	router, _ := upstream.LoadFromEnv(upstream.StubBackend{})
	srv := NewServer(gate, router, &audit.Recorder{})
	mux := http.NewServeMux()
	srv.Register(mux)

	soap := `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><FindFolder/></soap:Body></soap:Envelope>`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ews/Exchange.asmx", strings.NewReader(soap))
	req.Header.Set("SOAPAction", "FindFolder")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 got %d", rec.Code)
	}
}

func TestHealthzUpstreamModeSynthetic(t *testing.T) {
	t.Setenv("ERA_BRIDGE_SYNTHETIC", "1")
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsOutlookBridge})
	router, _ := upstream.LoadFromEnv(upstream.StubBackend{})
	srv := NewServer(gate, router, &audit.Recorder{})
	mux := http.NewServeMux()
	srv.Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"upstream_mode":"synthetic"`) {
		t.Fatalf("want upstream_mode=synthetic: %s", rec.Body.String())
	}
}

func TestBridgeProtocolUnauthorized(t *testing.T) {
	t.Setenv("ERA_BRIDGE_DEV", "")
	t.Setenv("ERA_PROTOCOL_BASIC_PASSWORD", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "")
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsOutlookBridge})
	router, _ := upstream.LoadFromEnv(upstream.StubBackend{})
	srv := NewServer(gate, router, &audit.Recorder{})
	mux := http.NewServeMux()
	srv.Register(mux)
	for _, path := range []string{"/ews/Exchange.asmx", "/caldav/"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: want 401 got %d", path, rec.Code)
		}
	}
}
