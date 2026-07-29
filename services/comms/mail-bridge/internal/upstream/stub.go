package upstream

import (
	"context"
	"fmt"
	"net/http"
)

// StubBackend returns 502 until a real adapter is configured.
type StubBackend struct{}

func (StubBackend) Name() string { return "stub" }

func (StubBackend) ProxyEWS(_ context.Context, _ string, _ []byte, _ http.Header) (int, []byte, error) {
	return http.StatusBadGateway, []byte(`<soap:Fault><faultstring>upstream not configured</faultstring></soap:Fault>`), fmt.Errorf("upstream stub")
}
