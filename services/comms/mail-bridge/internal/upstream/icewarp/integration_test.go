//go:build integration

package upstream_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	icewarpbe "era/services/comms/mail-bridge/internal/upstream/icewarp"
)

func TestIceWarpLabIntegration(t *testing.T) {
	base := os.Getenv("ERA_BRIDGE_TEST_ICEWARP_BASE_URL")
	if base == "" {
		t.Skip("ERA_BRIDGE_TEST_ICEWARP_BASE_URL not set")
	}
	be := icewarpbe.New(base)
	body := []byte(`<?xml version="1.0"?><soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><m:FindFolder xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages"/></soap:Body></soap:Envelope>`)
	status, resp, err := be.ProxyEWS(context.Background(), "FindFolder", body, http.Header{})
	if err != nil {
		t.Fatalf("proxy: %v", err)
	}
	if status >= 500 {
		t.Fatalf("status %d body %s", status, resp)
	}
	if len(resp) == 0 || !strings.Contains(string(resp), "soap") {
		t.Fatalf("unexpected response %s", resp)
	}
}
