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
