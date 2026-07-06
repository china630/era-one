package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/ingest-gateway/internal/grpcserver"
)

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	Routes(Config{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ожидался 200, получено %d", rec.Code)
	}
}

func TestIngestRejectsGet(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ingest", nil)
	Routes(Config{GRPC: grpcserver.New(nil)}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("ожидался 405, получено %d", rec.Code)
	}
}

func TestRBACStrictRejectsWithoutPrincipal(t *testing.T) {
	t.Setenv("ERA_RBAC_STRICT", "1")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/ingest", strings.NewReader(`{}`))
	Routes(Config{GRPC: grpcserver.New(nil)}).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ожидался 401, получено %d", rec.Code)
	}
}

func TestRBACStrictAllowsHealthWithoutPrincipal(t *testing.T) {
	t.Setenv("ERA_RBAC_STRICT", "1")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	Routes(Config{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидался 200, получено %d", rec.Code)
	}
}

func TestRBACPropagatesPrincipal(t *testing.T) {
	t.Setenv("ERA_RBAC_STRICT", "1")
	var got string
	h := WithRBAC(true, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("X-ERA-Principal", "svc-detection")
	h.ServeHTTP(rec, req)
	if got != "svc-detection" {
		t.Fatalf("principal=%q", got)
	}
	if rec.Header().Get("X-ERA-Principal") != "svc-detection" {
		t.Fatalf("response header not propagated")
	}
}

func TestIngestRejectsInvalidJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/ingest", strings.NewReader("not-json"))
	Routes(Config{GRPC: grpcserver.New(nil)}).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ожидался 400, получено %d", rec.Code)
	}
}
