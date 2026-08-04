package api_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/ai-core/internal/api"
	"era/services/ai-core/internal/investigate"
	"era/services/platform/licensegate"
)

// stubInv implements enough for license-deny path before CH (we use nil Inv carefully).
// Server checks gate before Inv — use BuildResult offline via New with nil Inv will panic on investigate.
// So we only hit gate by using FromModules(nil) and POST investigate.

func TestInvestigateDeniedWithoutAIModule(t *testing.T) {
	srv := api.New((*investigate.Client)(nil), licensegate.FromModules(nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/investigate", bytes.NewReader([]byte(`{"node_id":"n1"}`)))
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d %s", rec.Code, rec.Body.String())
	}
}
